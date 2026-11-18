import assert from "node:assert/strict"
import fs from "node:fs/promises"
import { setTimeout as delay } from "node:timers/promises"

const host = "http://127.0.0.1:18080"
const headers = { "Content-Type": "application/json" }
const userAgent = "xpanel-isolated-smoke"
headers["User-Agent"] = userAgent
const cookies = new Map()
async function request(path, method = "GET", body) {
    if (cookies.size) headers.Cookie = [...cookies].map(([key, value]) => key + "=" + value).join("; ")
    if (cookies.has("nz-csrf")) headers["X-CSRF-Token"] = cookies.get("nz-csrf")
    const response = await fetch(host + "/api/v1" + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) })
    for (const value of response.headers.getSetCookie()) {
        const pair = value.split(";", 1)[0]
        const i = pair.indexOf("=")
        cookies.set(pair.slice(0, i), pair.slice(i + 1))
    }
    const json = await response.json()
    assert(json.success, path + " " + json.error)
    return json.data
}
async function until(read, predicate) {
    const deadline = Date.now() + 45_000
    let value
    do {
        value = await read()
        if (predicate(value)) return value
        await delay(750)
    } while (Date.now() < deadline)
    throw new Error("timed out waiting for expected test state: " + JSON.stringify(value))
}

await request("/login", "POST", { username: "admin", password: "admin" })
const action = process.argv[2]
if (action === "registered") {
    const nodes = await until(() => request("/server"), (items) => items?.length === 2 && items.every((item) => item.node_role === "xpanel"))
    assert.equal(nodes.length, 2)
    console.log("Both real Agent processes registered as X-Panel nodes")
} else if (action === "apply") {
    const nodes = await request("/server")
    const batch = await request("/xpanel/appearance/apply", "POST", {
        server_ids: nodes.map((node) => node.id),
        profile: { schemaVersion: 1, themeId: "harbor", mode: "dark", accent: "#2F9284" },
    })
    const detail = await until(() => request("/xpanel/appearance/batches/" + batch.id),
        (value) => value.targets.every((target) => !["queued", "running"].includes(target.status)))
    const applied = detail.targets.find((target) => target.status === "applied")
    const offline = detail.targets.find((target) => target.status === "offline")
    assert(applied, JSON.stringify(detail.targets))
    assert(offline, JSON.stringify(detail.targets))
    await fs.writeFile("/state/batch.json", JSON.stringify({ batch: batch.id, offline: offline.server_id }))
    const history = await request("/xpanel/appearance/batches")
    assert.equal(history[0].id, batch.id)
    const attention = await request("/ops/attention")
    assert.equal(attention.appearance_nodes, 1)
    assert.equal(attention.appearance_batch_id, batch.id)
    console.log("Rollout persisted; online node applied and offline node appeared on the overview")
} else if (action === "retry") {
    const { batch, offline } = JSON.parse(await fs.readFile("/state/batch.json", "utf8"))
    const url = "/xpanel/appearance/batches/" + batch
    const prior = await request(url)
    assert(prior.targets.some((target) => target.server_id === offline && target.status === "offline"))
    await until(() => request("/server"), (items) => items?.some((node) =>
        node.id === offline && Date.now() - Date.parse(node.last_active) < 15_000))
    await request(url + "/retry/" + offline, "POST")
    const detail = await until(() => request(url), (value) => value.targets.every((target) => target.status === "applied"))
    assert.equal(detail.targets.find((target) => target.server_id === offline).attempts, 2)
    assert.equal(detail.targets.find((target) => target.server_id !== offline).attempts, 1)
    const attention = await request("/ops/attention")
    assert.equal(attention.appearance_nodes, 0)
    const layout = await fetch(host + "/dashboard/xpanel/appearance")
    assert.equal(layout.status, 200)
    assert.match(await layout.text(), /\/dashboard\/assets\//)
    await request("/logout", "POST")
    console.log("Dashboard restart preserved history; retry ran only on the chosen node and cleared the overview count")
} else if (action === "upgrade") {
    const { offline: nodeID } = JSON.parse(await fs.readFile("/state/batch.json", "utf8"))
    const tasks = await request("/xpanel/upgrade", "POST", { server_ids: [nodeID] })
    const results = await until(() => request("/xpanel/upgrade/tasks"), (rows) =>
        rows.some((row) => row.id === tasks[0].id && row.status === "success"))
    assert(results.find((row) => row.id === tasks[0].id).after_version)
    await request("/logout", "POST")
    console.log("Real Agent returned the simulated upgrade status through Dashboard")
}
