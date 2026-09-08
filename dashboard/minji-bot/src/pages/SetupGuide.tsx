import { useState } from "react"
import { Check, Copy, Download, PartyPopper, Rocket, ShieldCheck, Terminal } from "lucide-react"
import { Navbar } from "@/components/layout/Navbar"
import { Footer } from "@/components/layout/Footer"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { setupTemplates } from "@/data/setupTemplates"
import { downloadText } from "@/lib/utils"

function CodeBlock({ code, title, filename }: { code: string; title?: string; filename?: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div className="overflow-hidden rounded-none border border-border">
      {title && (
        <div className="flex items-center justify-between border-b border-border px-3 py-1.5">
          <span className="truncate text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{title}</span>
          <div className="flex shrink-0 items-center gap-2">
            {filename && (
              <button
                onClick={() => downloadText(filename, code)}
                className="flex items-center gap-1 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
                aria-label={`Download ${filename}`}
              >
                <Download className="size-3" />
                Download
              </button>
            )}
            <button
              onClick={copy}
              className="flex items-center gap-1 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
            >
              {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        </div>
      )}
      <pre className="max-h-72 overflow-auto px-3 py-2 font-mono text-xs leading-relaxed text-foreground">
        {code}
      </pre>
    </div>
  )
}

type ReferenceSection = { id: string; title: string; summary: string; snippet: string }

const REFERENCE: ReferenceSection[] = [
  {
    id: "server",
    title: "[server]",
    summary: "Renames the server. Both fields are optional.",
    snippet: `[server]
name = "My Community"
description = "A fresh server provisioned with MinjiBot."`,
  },
  {
    id: "roles",
    title: "[[roles]]",
    summary:
      "Creates a role. Colors are #RRGGBB. permissions take friendly names like send_messages, manage_messages, kick_members.",
    snippet: `[[roles]]
name = "Member"
color = "#5865f2"
permissions = ["view_channel", "send_messages", "read_message_history"]`,
  },
  {
    id: "channels",
    title: "[[channels]]",
    summary:
      "Creates a text, voice, category, announcement, or forum channel. parent references a category created earlier. permission_overwrites grant or deny per role/user.",
    snippet: `[[channels]]
name = "Lobby"
type = "category"

[[channels]]
name = "General"
type = "voice"
parent = "Lobby"

[[channels]]
name = "rules"
type = "text"
permission_overwrites = [
  { target = "role:Member", allow = ["view"], deny = ["send"] },
]`,
  },
  {
    id: "emojis",
    title: "[[emojis]]",
    summary: "Uploads an emoji from a public image URL (PNG/GIF/JPEG).",
    snippet: `[[emojis]]
name = "wave"
url = "https://example.com/wave.png"`,
  },
  {
    id: "stickers",
    title: "[[stickers]]",
    summary: "Uploads a sticker from a public image URL.",
    snippet: `[[stickers]]
name = "wave"
url = "https://example.com/wave.png"
description = "A friendly wave"
tags = "wave"`,
  },
  {
    id: "community",
    title: "[community]",
    summary:
      "Enables Community mode by pointing discord at the rules and announcements channels created above, and raises the verification level if you want.",
    snippet: `[community]
rules_channel = "rules"
updates_channel = "announcements"
verification_level = "low"`,
  },
  {
    id: "onboarding",
    title: "[onboarding]",
    summary:
      "Publishes the onboarding flow. Prompt options select roles and channels by name.",
    snippet: `[onboarding]
enabled = true
default_channels = ["chat"]

[[onboarding.prompts]]
title = "Who are you?"
required = true
single_select = true

[[onboarding.prompts.options]]
title = "Regular member"
roles = ["Member"]

[[onboarding.prompts.options]]
title = "Moderator"
roles = ["Moderator"]`,
  },
  {
    id: "commands",
    title: "[[commands]]",
    summary:
      "Runs a bot prefix command last, exactly as if a moderator typed it in the notify channel. {channel:Name} and {role:Name} become real mentions.",
    snippet: `[[commands]]
cmd = "channel"
args = "setperm {channel:chat} Member allow send"`,
  },
]

const STEPS = [
  {
    title: "Invite MinjiBot",
    body: "Add the bot to a fresh server. It needs Manage Roles, Manage Channels, and Manage Server for most documents.",
  },
  {
    title: "Open Server Setup",
    body: "Log in with Discord, pick the server from the dashboard, and open the Setup tab. You must hold Administrator — anything less is refused.",
  },
  {
    title: "Load a template or paste TOML",
    body: "Start from one of the templates below, upload a .toml file, or write your own with the reference to the right.",
  },
  {
    title: "Dry run",
    body: "Validates the document and checks that every role/channel name referenced later actually exists in the document.",
  },
  {
    title: "Run setup",
    body: "Applies the document top-to-bottom and reports each step. Fresh servers apply cleanly; established servers are refused.",
  },
]

export default function SetupGuide() {
  const [filter, setFilter] = useState("All")
  const templateFeatures = [...new Set(setupTemplates.flatMap((t) => t.features))]
  const visibleTemplates =
    filter === "All" ? setupTemplates : setupTemplates.filter((t) => t.features.includes(filter))

  return (
    <div className="min-h-screen bg-background font-sans antialiased">
      <Navbar />
      <main>
        <section className="border-b border-border bg-gradient-to-b from-primary/5 to-transparent">
          <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
            <Badge variant="outline" className="mb-4 gap-1.5">
              <Rocket className="size-3" />
              Server Setup
            </Badge>
            <h1 className="mb-4 max-w-2xl font-heading text-4xl font-bold tracking-tight text-foreground sm:text-5xl">
              Provision a server with a TOML document
            </h1>
            <p className="max-w-2xl text-muted-foreground">
              Declare the roles, channels, Community mode, onboarding flow, and setup commands once — MinjiBot
              applies them in order and reports every step. Here&apos;s how to use it, what the document looks like,
              and templates covering every feature, from a minimal server to Community mode, forums, and onboarding.
            </p>
          </div>
        </section>

        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
          <div className="grid gap-10 lg:grid-cols-[1fr_380px]">
            <div className="min-w-0 space-y-10">
              <section>
                <h2 className="mb-4 font-heading text-2xl font-semibold tracking-tight text-foreground">How it works</h2>
                <ol className="space-y-4">
                  {STEPS.map((step, i) => (
                    <li key={step.title} className="flex gap-4">
                      <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                        {i + 1}
                      </span>
                      <div>
                        <p className="font-medium text-foreground">{step.title}</p>
                        <p className="text-sm text-muted-foreground">{step.body}</p>
                      </div>
                    </li>
                  ))}
                </ol>
              </section>

              <section>
                <div className="mb-4 flex items-center gap-2">
                  <ShieldCheck className="size-5 text-foreground" />
                  <h2 className="font-heading text-2xl font-semibold tracking-tight text-foreground">Safety</h2>
                </div>
                <Card>
                  <CardContent className="space-y-3 pt-6">
                    <p className="text-sm text-muted-foreground">
                      Setup only runs when <strong className="text-foreground">you hold Administrator</strong> in the
                      target server — enforced by Discord&apos;s own permission bits, not by the app.
                    </p>
                    <p className="text-sm text-muted-foreground">
                      And because provisioning creates roles, channels, and settings, servers{" "}
                      <strong className="text-foreground">that already look live are refused</strong>: a built-out
                      combination of channels and third-party bots, or more than 20 members, trips the guard. Setup is
                      for fresh servers.
                    </p>
                  </CardContent>
                </Card>
              </section>

              <section>
                <div className="mb-4 flex items-center gap-2">
                  <PartyPopper className="size-5 text-foreground" />
                  <h2 className="font-heading text-2xl font-semibold tracking-tight text-foreground">Templates</h2>
                </div>
                <div className="mb-4 flex flex-wrap items-center gap-2">
                  {["All", ...templateFeatures].map((feature) => (
                    <button
                      key={feature}
                      onClick={() => setFilter(feature)}
                      className={
                        "rounded-full border px-3 py-1 text-xs font-medium transition-colors " +
                        (filter === feature
                          ? "border-primary bg-primary text-primary-foreground"
                          : "border-border text-muted-foreground hover:border-primary hover:text-foreground")
                      }
                    >
                      {feature}
                    </button>
                  ))}
                </div>
                <div className="space-y-4">
                  {visibleTemplates.map((tpl) => (
                    <Card key={tpl.id}>
                      <CardHeader>
                        <CardTitle>{tpl.name}</CardTitle>
                        <CardDescription>{tpl.description}</CardDescription>
                        <div className="flex flex-wrap gap-1.5 pt-1">
                          {tpl.features.map((feature) => (
                            <Badge key={feature} variant="outline" className="text-[11px] font-normal">
                              {feature}
                            </Badge>
                          ))}
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
                        <CodeBlock code={tpl.toml} title={`${tpl.name} — setup.toml`} filename={`setup-${tpl.id}.toml`} />
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-xs text-muted-foreground">
                            Paste into the Setup tab, then Dry run.
                          </span>
                          <a
                            href="/dashboard"
                            className={buttonVariants({ variant: "outline", size: "sm" }) + " gap-1.5"}
                          >
                            Take me to the dashboard
                          </a>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </section>
            </div>

            <aside className="min-w-0 space-y-6 lg:sticky lg:top-6 lg:self-start">
              <section>
                <div className="mb-3 flex items-center gap-2">
                  <Terminal className="size-4 text-muted-foreground" />
                  <h2 className="font-heading text-lg font-semibold tracking-tight text-foreground">Document reference</h2>
                </div>
                <nav className="space-y-0.5">
                  {REFERENCE.map((ref) => (
                    <a
                      key={ref.id}
                      href={`#${ref.id}`}
                      className="flex items-center gap-2 rounded-none border-l-2 border-transparent px-2 py-1 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-foreground"
                    >
                      {ref.title.replace(/[[\]]/g, "")}
                    </a>
                  ))}
                </nav>
              </section>

              {REFERENCE.map((ref) => (
                <section key={ref.id} id={ref.id} className="scroll-mt-6">
                  <h3 className="mb-1 font-mono text-sm font-semibold text-foreground">{ref.title}</h3>
                  <p className="mb-2 text-xs text-muted-foreground">{ref.summary}</p>
                  <CodeBlock code={ref.snippet} />
                </section>
              ))}

              <Card>
                <CardContent className="pt-6">
                  <p className="text-sm text-muted-foreground">
                    Everything is optional — a document with just <code className="font-mono text-xs">[server]</code>{" "}
                    and a single channel is a valid setup file.
                  </p>
                </CardContent>
              </Card>
            </aside>
          </div>
        </div>
      </main>
      <Footer />
    </div>
  )
}