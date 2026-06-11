# Routing Rule Sources

This project keeps a small curated routing rule set in code instead of importing
remote rule providers. The goal is stable cross-client behavior for Clash/Mihomo,
Surge, and Shadowrocket without making user subscriptions depend on GitHub raw
or CDN availability.

## Reference Projects

- blackmatrix7/ios_rule_script: https://github.com/blackmatrix7/ios_rule_script
  - Broad client ecosystem coverage: Clash, Surge, Quantumult X, Shadowrocket.
  - Useful as a classification reference for common service families.
  - License observed during planning: GPL-2.0.
- MetaCubeX/meta-rules-dat: https://github.com/MetaCubeX/meta-rules-dat
  - Useful reference for Mihomo/Clash geosite-lite categories such as OpenAI,
    YouTube, Netflix, GitHub, Telegram, Microsoft, Apple, and CN.
  - License observed during planning: GPL-3.0.
- Loyalsoldier/clash-rules: https://github.com/Loyalsoldier/clash-rules
  - Useful reference for base layers such as direct, proxy, reject, private,
    Apple, and Google.
  - License observed during planning: GPL-3.0.
- Loyalsoldier/v2ray-rules-dat: https://github.com/Loyalsoldier/v2ray-rules-dat
  - Useful reference for whitelist/blacklist routing ideas, ads, CN, and
    geolocation-!cn style splitting.
  - License observed during planning: GPL-3.0.
- v2fly/domain-list-community: https://github.com/v2fly/domain-list-community
  - Useful neutral domain categorization reference.
  - Its project goal is classification, not deciding whether a domain should be
    blocked or proxied.

## Local Policy

- Do not wholesale copy large GPL rule lists into the codebase.
- Use the projects above as references, then manually curate a small set of
  high-signal domains for this service.
- Keep rule behavior deterministic and generated from one source of truth.
- Keep AI routing business-aware: AI domains target residential chained egress
  when chain proxy configuration exists; otherwise they fall back to the normal
  Auto group instead of generating broken client rules.
- Keep unknown traffic simple: fall through to Auto.

## Rule Order

The subscription generators keep this priority:

1. Reject
2. AI
3. Streaming
4. Global
5. ChinaDirect
6. GEOIP CN
7. Fallback Auto

