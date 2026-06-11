import assert from "node:assert/strict"
import { parseDeployOutput } from "../src/utils/nodeImport.ts"

const sample = `
plain node:
  Name: JP4-plain
  Host: 120.231.184.128
  Port: 443
  SNI: bing.com
  Password: 留空
  Skip Cert Verify: 勾选
  Obfs Type: None / 留空
  Obfs Password: 留空
  Traffic API URL: http://120.231.184.128:25413
  Traffic API Secret: f361798f168025f16f18ea152f21adb2
  Sort Order: 10

obfs node:
  Name: JP4-obfs
  Host: 120.231.184.128
  Port: 8443
  SNI: bing.com
  Password: 留空
  Skip Cert Verify: 勾选
  Obfs Type: salamander
  Obfs Password: 64248ad1e1d9a393a831a1fd0b5b7f62
  Traffic API URL: http://120.231.184.128:25414
  Traffic API Secret: 03c0b451506882b77377955209c7c296
  Sort Order: 11
`

const nodes = parseDeployOutput(sample)
assert.equal(nodes.length, 2)
assert.deepEqual(nodes[0], {
  name: "JP4-plain",
  host: "120.231.184.128",
  port: 443,
  password: "",
  sni: "bing.com",
  insecure: true,
  obfs_type: "",
  obfs_password: "",
  traffic_api: "http://120.231.184.128:25413",
  traffic_secret: "f361798f168025f16f18ea152f21adb2",
  sort_order: 10,
})
assert.deepEqual(nodes[1], {
  name: "JP4-obfs",
  host: "120.231.184.128",
  port: 8443,
  password: "",
  sni: "bing.com",
  insecure: true,
  obfs_type: "salamander",
  obfs_password: "64248ad1e1d9a393a831a1fd0b5b7f62",
  traffic_api: "http://120.231.184.128:25414",
  traffic_secret: "03c0b451506882b77377955209c7c296",
  sort_order: 11,
})

assert.throws(() => parseDeployOutput("Name: broken"), /No complete node blocks/)

console.log("node import parser test passed")
