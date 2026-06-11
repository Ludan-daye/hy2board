package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// PaymentInput is the body block embedded inside RenewRequest / ApplyPlan.
// Absent → no Payment row written. Present → 1 row created in same tx.
type PaymentInput struct {
	AmountCents int64      `json:"amount_cents"`
	Kind        string     `json:"kind"` // "new" | "renew"; if empty, auto-detected
	Note        string     `json:"note"`
	PaidAt      *time.Time `json:"paid_at,omitempty"` // nil → now()
}

// CreatePayment writes one row inside the given GORM tx. Caller passes the
// effective userID, planID, daysAdded, operator (admin username).
// kind is auto-detected if input.Kind is empty: any prior row (incl. soft-deleted)
// for this user → "renew", else "new".
func CreatePayment(tx *gorm.DB, userID, planID uint, daysAdded int, operator string, in PaymentInput) error {
	kind := in.Kind
	if kind != "new" && kind != "renew" {
		var c int64
		tx.Unscoped().Model(&model.Payment{}).Where("user_id = ?", userID).Count(&c)
		if c == 0 {
			kind = "new"
		} else {
			kind = "renew"
		}
	}
	paidAt := time.Now().UTC()
	if in.PaidAt != nil {
		paidAt = in.PaidAt.UTC()
	}
	p := model.Payment{
		UserID:      userID,
		PlanID:      planID,
		AmountCents: in.AmountCents,
		DaysAdded:   daysAdded,
		Kind:        kind,
		Note:        in.Note,
		Operator:    operator,
		PaidAt:      paidAt,
	}
	return tx.Create(&p).Error
}

// adminOperator extracts the logged-in admin username from gin context
// (set by middleware.AdminAuth as c.Set("admin", username)).
func adminOperator(getter interface{ GetString(string) string }) string {
	return getter.GetString("admin")
}

// dbForCtx returns the global DB; placeholder so handlers can later swap to a
// request-scoped tx if needed.
func dbForCtx() *gorm.DB { return database.DB }

type paymentRow struct {
	ID          uint      `json:"ID"`
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	PlanID      uint      `json:"plan_id"`
	PlanName    string    `json:"plan_name"`
	AmountCents int64     `json:"amount_cents"`
	DaysAdded   int       `json:"days_added"`
	Kind        string    `json:"kind"`
	Note        string    `json:"note"`
	Operator    string    `json:"operator"`
	PaidAt      time.Time `json:"paid_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func ListPayments(c *gin.Context) {
	q := database.DB.Table("payments").
		Select(`payments.id AS ID, payments.user_id, users.username,
		        payments.plan_id, COALESCE(plans.name,'') AS plan_name,
		        payments.amount_cents, payments.days_added, payments.kind,
		        payments.note, payments.operator, payments.paid_at, payments.created_at`).
		Joins("LEFT JOIN users ON users.id = payments.user_id").
		Joins("LEFT JOIN plans ON plans.id = payments.plan_id").
		Where("payments.deleted_at IS NULL")

	if v := c.Query("user_id"); v != "" {
		q = q.Where("payments.user_id = ?", v)
	}
	if v := c.Query("plan_id"); v != "" {
		q = q.Where("payments.plan_id = ?", v)
	}
	if v := c.Query("from"); v != "" {
		q = q.Where("payments.paid_at >= ?", v)
	}
	if v := c.Query("to"); v != "" {
		q = q.Where("payments.paid_at < ?", v+" 23:59:59")
	}

	var total int64
	q.Count(&total)

	page := 1
	size := 50
	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("size"); v != "" {
		fmt.Sscanf(v, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}

	var items []paymentRow
	q.Order("payments.paid_at DESC").Limit(size).Offset((page - 1) * size).Scan(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "size": size})
}

type monthBucket struct {
	Month       string `json:"month"`
	TotalCents  int64  `json:"total_cents"`
	CostCents   int64  `json:"cost_cents"`
	ProfitCents int64  `json:"profit_cents"`
	Count       int    `json:"count"`
	CostCount   int    `json:"cost_count"`
	NewCount    int    `json:"new_count"`
	RenewCount  int    `json:"renew_count"`
}

func SummaryPayments(c *gin.Context) {
	n := 12
	if v := c.Query("n"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}
	if n < 1 || n > 60 {
		n = 12
	}

	now := time.Now().UTC()
	// Build n month keys from oldest to current.
	keys := make([]string, 0, n)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := n - 1; i >= 0; i-- {
		m := startOfMonth.AddDate(0, -i, 0)
		keys = append(keys, m.Format("2006-01"))
	}

	type rawRow struct {
		Month      string
		TotalCents int64
		Count      int
		NewCount   int
		RenewCount int
	}
	type rawCostRow struct {
		Month     string
		CostCents int64
		CostCount int
	}
	var raws []rawRow
	cutoff := startOfMonth.AddDate(0, -(n - 1), 0).Format("2006-01-02")
	database.DB.Raw(`
		SELECT strftime('%Y-%m', paid_at) AS month,
		       COALESCE(SUM(amount_cents),0) AS total_cents,
		       COUNT(*) AS count,
		       SUM(CASE WHEN kind='new' THEN 1 ELSE 0 END) AS new_count,
		       SUM(CASE WHEN kind='renew' THEN 1 ELSE 0 END) AS renew_count
		FROM payments
		WHERE deleted_at IS NULL AND paid_at >= ?
		GROUP BY month`, cutoff).Scan(&raws)

	var costRaws []rawCostRow
	database.DB.Raw(`
		SELECT strftime('%Y-%m', incurred_at) AS month,
		       COALESCE(SUM(amount_cents),0) AS cost_cents,
		       COUNT(*) AS cost_count
		FROM costs
		WHERE deleted_at IS NULL AND incurred_at >= ?
		GROUP BY month`, cutoff).Scan(&costRaws)

	bucketMap := map[string]rawRow{}
	for _, r := range raws {
		bucketMap[r.Month] = r
	}
	costMap := map[string]rawCostRow{}
	for _, r := range costRaws {
		costMap[r.Month] = r
	}

	out := make([]monthBucket, 0, n)
	for _, k := range keys {
		r := bucketMap[k]
		cost := costMap[k]
		out = append(out, monthBucket{
			Month: k, TotalCents: r.TotalCents, CostCents: cost.CostCents,
			ProfitCents: r.TotalCents - cost.CostCents,
			Count:       r.Count, CostCount: cost.CostCount,
			NewCount: r.NewCount, RenewCount: r.RenewCount,
		})
	}
	c.JSON(http.StatusOK, out)
}

type updatePaymentRequest struct {
	AmountCents *int64     `json:"amount_cents,omitempty"`
	Kind        *string    `json:"kind,omitempty"`
	Note        *string    `json:"note,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	DaysAdded   *int       `json:"days_added,omitempty"`
}

func UpdatePayment(c *gin.Context) {
	var p model.Payment
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	var req updatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.AmountCents != nil {
		updates["amount_cents"] = *req.AmountCents
	}
	if req.Kind != nil {
		updates["kind"] = *req.Kind
	}
	if req.Note != nil {
		updates["note"] = *req.Note
	}
	if req.PaidAt != nil {
		updates["paid_at"] = (*req.PaidAt).UTC()
	}
	if req.DaysAdded != nil {
		updates["days_added"] = *req.DaysAdded
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, p)
		return
	}
	if err := database.DB.Model(&p).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&p, p.ID)
	auditAdminAction(c, "payment.update", "payment", p.ID, "", map[string]interface{}{
		"fields":  auditKeys(updates),
		"user_id": p.UserID,
		"plan_id": p.PlanID,
	})
	c.JSON(http.StatusOK, p)
}

func DeletePayment(c *gin.Context) {
	var p model.Payment
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	database.DB.Delete(&p)
	auditAdminAction(c, "payment.delete", "payment", p.ID, "", map[string]interface{}{
		"user_id": p.UserID,
		"plan_id": p.PlanID,
		"amount":  p.AmountCents,
	})
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ExportPaymentsCSV(c *gin.Context) {
	q := database.DB.Table("payments").
		Select(`payments.paid_at, COALESCE(users.username,'') AS username,
		        COALESCE(plans.name,'') AS plan_name,
		        payments.amount_cents, payments.days_added,
		        payments.kind, payments.note, payments.operator`).
		Joins("LEFT JOIN users ON users.id = payments.user_id").
		Joins("LEFT JOIN plans ON plans.id = payments.plan_id").
		Where("payments.deleted_at IS NULL")
	if v := c.Query("from"); v != "" {
		q = q.Where("payments.paid_at >= ?", v)
	}
	if v := c.Query("to"); v != "" {
		q = q.Where("payments.paid_at < ?", v+" 23:59:59")
	}
	if v := c.Query("user_id"); v != "" {
		q = q.Where("payments.user_id = ?", v)
	}
	if v := c.Query("plan_id"); v != "" {
		q = q.Where("payments.plan_id = ?", v)
	}

	type csvRow struct {
		PaidAt      time.Time
		Username    string
		PlanName    string
		AmountCents int64
		DaysAdded   int
		Kind        string
		Note        string
		Operator    string
	}
	var rows []csvRow
	q.Order("payments.paid_at DESC").Scan(&rows)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=payments-"+time.Now().UTC().Format("2006-01-02")+".csv")
	w := csv.NewWriter(c.Writer)
	w.Write([]string{"paid_at", "username", "plan_name", "amount_cny", "days_added", "kind", "note", "operator"})
	for _, r := range rows {
		yuan := strconv.FormatFloat(float64(r.AmountCents)/100.0, 'f', 2, 64)
		w.Write([]string{
			r.PaidAt.UTC().Format("2006-01-02 15:04:05"),
			r.Username, r.PlanName, yuan,
			strconv.Itoa(r.DaysAdded), r.Kind, r.Note, r.Operator,
		})
	}
	w.Flush()
}
