// Parses the REGISTER blocks printed by vless-pilot-setup.sh and trojan-add.sh.
// Handles both multi-line "key = value" and inline "key=value" forms.
export interface NodeRegisterFields {
  vless_enabled?: boolean
  vless_port?: number
  reality_pubkey?: string
  reality_shortid?: string
  reality_sni?: string
  vless_stats_api?: string
  vless_stats_secret?: string
  trojan_enabled?: boolean
  trojan_port?: number
  trojan_sni?: string
}

function grab(text: string, key: string): string | undefined {
  // key, optional spaces, =, optional spaces, then a non-space token (quotes stripped)
  const m = text.match(new RegExp(`(?:^|\\s)${key}\\s*=\\s*"?([^\\s"]+)"?`, "i"))
  return m ? m[1] : undefined
}

export function parseRegister(text: string): NodeRegisterFields {
  const out: NodeRegisterFields = {}
  const str = (k: string, set: (v: string) => void) => { const v = grab(text, k); if (v) set(v) }
  const num = (k: string, set: (v: number) => void) => { const v = grab(text, k); if (v && /^\d+$/.test(v)) set(parseInt(v, 10)) }
  const bool = (k: string, set: (v: boolean) => void) => { const v = grab(text, k); if (v) set(v === "1" || v.toLowerCase() === "true") }

  bool("vless_enabled", v => out.vless_enabled = v)
  num("vless_port", v => out.vless_port = v)
  str("reality_pubkey", v => out.reality_pubkey = v)
  str("reality_shortid", v => out.reality_shortid = v)
  str("reality_sni", v => out.reality_sni = v)
  str("vless_stats_api", v => out.vless_stats_api = v)
  str("vless_stats_secret", v => out.vless_stats_secret = v)
  bool("trojan_enabled", v => out.trojan_enabled = v)
  num("trojan_port", v => out.trojan_port = v)
  str("trojan_sni", v => out.trojan_sni = v)
  return out
}
