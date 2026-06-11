export interface NodeImportForm {
  name: string
  host: string
  port: number
  password: string
  sni: string
  insecure: boolean
  obfs_type: string
  obfs_password: string
  traffic_api: string
  traffic_secret: string
  sort_order: number
}

type NodeFields = Record<string, string>

const fieldAliases: Record<string, keyof NodeFields> = {
  name: "name",
  host: "host",
  port: "port",
  sni: "sni",
  password: "password",
  "skip cert verify": "insecure",
  "obfs type": "obfs_type",
  "obfs password": "obfs_password",
  "traffic api url": "traffic_api",
  "traffic api secret": "traffic_secret",
  "sort order": "sort_order",
}

function emptyish(value: string): boolean {
  const v = value.trim().toLowerCase()
  return v === "" || v === "-" || v === "none" || v === "none / 留空" || v === "留空" || v === "无"
}

function cleanOptional(value: string): string {
  return emptyish(value) ? "" : value.trim()
}

function parseBool(value: string): boolean {
  return /^(true|1|yes|y|on|enabled|勾选|是)$/i.test(value.trim())
}

function isCompleteBlock(fields: NodeFields): boolean {
  return Boolean(fields.name && fields.host && fields.port && fields.traffic_api && fields.traffic_secret)
}

function toNode(fields: NodeFields): NodeImportForm {
  if (!isCompleteBlock(fields)) {
    throw new Error("Incomplete node block")
  }
  const port = Number.parseInt(fields.port, 10)
  const sortOrder = Number.parseInt(fields.sort_order || "0", 10)
  if (!Number.isFinite(port) || port <= 0) {
    throw new Error(`Invalid port for ${fields.name}`)
  }
  if (!Number.isFinite(sortOrder)) {
    throw new Error(`Invalid sort order for ${fields.name}`)
  }
  return {
    name: fields.name.trim(),
    host: fields.host.trim(),
    port,
    password: cleanOptional(fields.password || ""),
    sni: cleanOptional(fields.sni || "bing.com") || "bing.com",
    insecure: parseBool(fields.insecure || ""),
    obfs_type: cleanOptional(fields.obfs_type || ""),
    obfs_password: cleanOptional(fields.obfs_password || ""),
    traffic_api: fields.traffic_api.trim(),
    traffic_secret: fields.traffic_secret.trim(),
    sort_order: sortOrder,
  }
}

export function parseDeployOutput(text: string): NodeImportForm[] {
  const blocks: NodeFields[] = []
  let current: NodeFields | null = null

  const finish = () => {
    if (current && Object.keys(current).length > 0) {
      blocks.push(current)
    }
    current = null
  }

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (/^(plain|obfs)\s+node\s*:/i.test(line)) {
      finish()
      current = {}
      continue
    }

    const match = rawLine.match(/^\s*([A-Za-z][A-Za-z ]+):\s*(.*?)\s*$/)
    if (!match) {
      continue
    }
    const label = match[1].trim().toLowerCase()
    const key = fieldAliases[label]
    if (!key) {
      continue
    }

    if (!current && label === "name") {
      current = {}
    }
    if (current) {
      current[key] = match[2].trim()
    }
  }
  finish()

  const nodes = blocks.filter(isCompleteBlock).map(toNode)
  if (nodes.length === 0) {
    throw new Error("No complete node blocks found")
  }
  return nodes
}
