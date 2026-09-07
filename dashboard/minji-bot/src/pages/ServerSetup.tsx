import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { CheckCircle2, CircleOff, FileText, Loader2, Play, Rocket, Upload, XCircle } from "lucide-react"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { apiUrl } from "@/lib/api"
import { useCurrentUser } from "@/lib/useCurrentUser"

type Channel = { id: string; name: string; type: number; parent_id: string }

type Step = {
  section: string
  label: string
  status: "ok" | "failed" | "skipped"
  detail: string
}

const SAMPLE_TOML = `# MinjiBot server setup
# Applied top-to-bottom: server, roles, channels, emojis, stickers,
# community (enables Community mode), onboarding, then robot commands.

[server]
name = "Neroville"

[[roles]]
name = "Member"
color = "#7289da"
permissions = ["view_channel", "send_messages", "read_message_history", "add_reactions"]

[[roles]]
name = "Moderator"
color = "#ed4245"
permissions = ["manage_messages", "kick_members", "ban_members", "manage_roles", "timeout_members"]

[[channels]]
name = "info"
type = "text"
topic = "Welcome to the server!"

[[channels]]
name = "chat"
type = "text"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [ { target = "role:Member", allow = ["view", "send"], deny = [] } ]

[[channels]]
name = "Voice"
type = "category"

[[channels]]
name = "General"
type = "voice"
parent = "Voice"

[[emojis]]
name = "kibet"
url = "https://cdn.discordapp.com/emojis/937248503783063575.png"

[[stickers]]
name = "wave"
url = "https://cdn.discordapp.com/attachments/937248503783063575/937248503783063575/wave.png"
description = "A friendly wave"

[community]
rules_channel = "rules"
updates_channel = "info"
verification_level = "low"
content_filter = "members_without_roles"

[onboarding]
enabled = true
default_channels = ["chat"]

[[onboarding.prompts]]
title = "Who are you?"
required = true
single_select = true

[[onboarding.prompts.options]]
title = "Regular"
roles = ["Member"]

[[onboarding.prompts.options]]
title = "Moderator"
roles = ["Moderator"]

# Commands run last so they can reference channels/roles created above.
# {channel:chat} and {role:Member} turn into real Discord mentions.
[[commands]]
cmd = "channel"
args = "setperm {channel:chat} Member allow send"`

function statusVariant(status: Step["status"]): "default" | "destructive" | "secondary" | "outline" {
  if (status === "failed") return "destructive"
  if (status === "skipped") return "secondary"
  return "default"
}

function StatusIcon({ status }: { status: Step["status"] }) {
  if (status === "failed") return <XCircle className="size-4 text-destructive" />
  if (status === "skipped") return <CircleOff className="size-4 text-muted-foreground" />
  return <CheckCircle2 className="size-4 text-emerald-500" />
}

export default function ServerSetup() {
  const { guildId } = useParams<{ guildId: string }>()
  const me = useCurrentUser()

  const [channels, setChannels] = useState<Channel[] | null>(null)
  const [notifyChannel, setNotifyChannel] = useState("")
  const [toml, setToml] = useState("")
  const [results, setResults] = useState<Step[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<null | "dry" | "run">(null)

  useEffect(() => {
    if (me.status !== "authenticated" || !guildId) return
    let cancelled = false
    fetch(apiUrl(`/api/guilds/${guildId}/channels`), {
      credentials: "include",
      headers: { Accept: "application/json" },
    })
      .then(async (res) => {
        if (!res.ok) throw new Error(`${res.status}`)
        return (await res.json()) as Channel[]
      })
      .then((data) => {
        if (!cancelled) {
          setChannels(data)
          const text = data.find((c) => c.type === 0) ?? data[0]
          if (text) setNotifyChannel(text.id)
        }
      })
      .catch((err) => {
        console.error("channels:", err)
        if (!cancelled) setError("Could not load channels.")
      })
    return () => {
      cancelled = true
    }
  }, [me.status, guildId])

  function handleFile(file: File | undefined) {
    if (!file) return
    file.text().then((text) => setToml(text)).catch(() => setError("Could not read that file."))
  }

  async function handleValidate() {
    if (!guildId || !toml.trim()) {
      setError("Paste or upload a TOML document first.")
      return
    }
    setBusy("dry")
    setError(null)
    setResults(null)
    try {
      const res = await fetch(apiUrl(`/api/guilds/${guildId}/setup/dry-run`), {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ toml }),
      })
      const data = (await res.json().catch(() => null)) ?? {}
      if (!res.ok) {
        throw new Error((data as { error?: string }).error ?? `HTTP ${res.status}`)
      }
      setResults([{ section: "validation", label: "dry run", status: "ok", detail: "document is valid and references resolve" }])
    } catch (err) {
      setError(String(err))
    } finally {
      setBusy(null)
    }
  }

  async function handleRun() {
    if (!guildId || !toml.trim()) {
      setError("Paste or upload a TOML document first.")
      return
    }
    setBusy("run")
    setError(null)
    setResults(null)
    try {
      const res = await fetch(apiUrl(`/api/guilds/${guildId}/setup`), {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ toml, notify_channel_id: notifyChannel }),
      })
      const data = (await res.json().catch(() => null)) ?? {}
      if (!res.ok) {
        throw new Error((data as { error?: string }).error ?? `HTTP ${res.status}`)
      }
      setResults((data as { steps: Step[] }).steps ?? [])
    } catch (err) {
      setError(String(err))
    } finally {
      setBusy(null)
    }
  }

  const failedCount = results?.filter((s) => s.status === "failed").length ?? 0

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <div className="mb-6">
        <h1 className="mb-1 font-heading text-3xl font-bold tracking-tight text-foreground">
          Server setup
        </h1>
        <p className="text-sm text-muted-foreground">
          Provision a server from a TOML document: roles, channels, emojis, stickers,
          Community mode, onboarding, and commands. Only you — an Administrator — can run this.
        </p>
      </div>

      {me.status !== "authenticated" ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-sm text-muted-foreground">Log in to run a server setup.</p>
            <a href={apiUrl("/api/auth/discord")} className={buttonVariants({ variant: "default" }) + " mt-4"}>
              Log in with Discord
            </a>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Document</CardTitle>
              <CardDescription>
                Paste TOML, upload a <code className="font-mono text-xs">.toml</code> file, or load the sample below.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <label className="inline-flex">
                  <input
                    type="file"
                    accept=".toml,.tml,text/plain"
                    className="sr-only"
                    onChange={(e) => handleFile(e.target.files?.[0])}
                  />
                  <span className={buttonVariants({ variant: "outline", size: "sm" }) + " cursor-pointer gap-1.5"}>
                    <Upload className="size-3.5" />
                    Upload .toml
                  </span>
                </label>
                <Button variant="ghost" size="sm" onClick={() => setToml(SAMPLE_TOML)} className="gap-1.5">
                  <FileText className="size-3.5" />
                  Load sample
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setToml("")} disabled={!toml}>
                  Clear
                </Button>
              </div>

              <textarea
                value={toml}
                onChange={(e) => setToml(e.target.value)}
                rows={18}
                placeholder={`[server]\nname = "My Server"\n\n[[roles]]\nname = "Member"\npermissions = ["send_messages"]\n\n[[channels]]\nname = "general"\ntype = "text"`}
                spellCheck={false}
                className="w-full resize-y rounded-none border border-input bg-transparent px-3 py-2 font-mono text-xs outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/50"
              />

              <div className="grid gap-4 sm:grid-cols-[280px_1fr]">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium" htmlFor="notifyChannel">
                    Notify channel
                  </label>
                  {channels === null ? (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Loader2 className="size-3.5 animate-spin" />
                      Loading channels…
                    </div>
                  ) : (
                    <select
                      id="notifyChannel"
                      value={notifyChannel}
                      onChange={(e) => setNotifyChannel(e.target.value)}
                      className="h-8 w-full rounded-none border border-input bg-transparent px-2.5 py-1 text-xs outline-none focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/50"
                    >
                      {channels.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </select>
                  )}
                  <p className="text-xs text-muted-foreground">
                    Where command replies and status messages are posted.
                  </p>
                </div>

                <div className="flex items-end gap-3">
                  <Button variant="outline" onClick={handleValidate} disabled={busy !== null}>
                    {busy === "dry" ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle2 className="size-4" />}
                    {busy === "dry" ? "Checking…" : "Dry run"}
                  </Button>
                  <Button onClick={handleRun} disabled={busy !== null} className="gap-1.5">
                    {busy === "run" ? <Loader2 className="size-4 animate-spin" /> : <Rocket className="size-4" />}
                    {busy === "run" ? "Running…" : "Run setup"}
                  </Button>
                  {busy === "run" && (
                    <span className="text-xs text-muted-foreground">
                      <Play className="mr-1 inline size-3" />
                      applying steps in order…
                    </span>
                  )}
                </div>
              </div>

              {error && <p className="text-xs text-destructive">{error}</p>}
            </CardContent>
          </Card>

          {results && results.length > 0 && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>Results</CardTitle>
                  {failedCount > 0 ? (
                    <Badge variant="destructive">{failedCount} failed</Badge>
                  ) : (
                    <Badge variant="outline">all ok</Badge>
                  )}
                </div>
                <CardDescription>
                  {results.length} step{results.length === 1 ? "" : "s"}{" "}
                  {busy === "run" ? "applied so far" : "recorded"}.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-1.5">
                  {results.map((step, i) => (
                    <div key={i} className="flex items-start gap-2 rounded-none border border-border px-3 py-2 text-xs">
                      <StatusIcon status={step.status} />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium text-foreground">
                            {step.section} &middot; {step.label}
                          </span>
                          <Badge variant={statusVariant(step.status)}>{step.status}</Badge>
                        </div>
                        {step.detail && (
                          <p className="mt-0.5 font-mono text-[11px] break-words text-muted-foreground">{step.detail}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  )
}