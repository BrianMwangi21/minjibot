import { useEffect, useState } from "react"
import { Link, NavLink, Outlet, useNavigate, useParams } from "react-router-dom"
import { BookOpen, LayoutDashboard, Loader2, Rocket, ScrollText, Settings, User } from "lucide-react"
import { DashboardHeader } from "@/components/dashboard/DashboardHeader"
import { apiUrl } from "@/lib/api"
import { useCurrentUser } from "@/lib/useCurrentUser"
import { cn } from "@/lib/utils"

type GuildSummary = { id: string; name: string }

const GUILD_LINKS = (guildId: string) => [
  { to: `/dashboard/guild/${guildId}`, label: "Logs", icon: ScrollText, end: true },
  { to: `/dashboard/guild/${guildId}/settings`, label: "Settings", icon: Settings, end: false },
  { to: `/dashboard/guild/${guildId}/setup`, label: "Setup", icon: Rocket, end: false },
]

// GuildLayout surrounds every per-guild dashboard page with a persistent
// sidebar: a server switcher on top, then the pages that apply to that server.
// The top-right links (Diary, Profile) keep to the user's own use of the app.
export function GuildLayout() {
  const { guildId } = useParams<{ guildId: string }>()
  const me = useCurrentUser()
  const navigate = useNavigate()
  const [guilds, setGuilds] = useState<GuildSummary[] | null>(null)

  useEffect(() => {
    if (me.status !== "authenticated") return
    let cancelled = false
    fetch(apiUrl("/api/guilds"), { credentials: "include", headers: { Accept: "application/json" } })
      .then(async (res) => {
        if (!res.ok) throw new Error(`${res.status}`)
        return (await res.json()) as GuildSummary[]
      })
      .then((data) => {
        if (!cancelled) setGuilds(data)
      })
      .catch(() => {
        if (!cancelled) setGuilds([])
      })
    return () => {
      cancelled = true
    }
  }, [me.status])

  const current = guilds?.find((g) => g.id === guildId)
  const links = GUILD_LINKS(guildId ?? "")

  return (
    <div className="min-h-screen bg-background font-sans antialiased">
      <DashboardHeader />
      <div className="mx-auto flex max-w-7xl gap-6 px-4 py-6 sm:px-6">
        <aside className="sticky top-6 hidden h-fit w-60 shrink-0 lg:block">
          <div className="mb-5">
            <p className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Server
            </p>
            {guilds === null ? (
              <div className="flex items-center gap-2 px-2 text-xs text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" />
                Loading servers…
              </div>
            ) : guilds.length === 0 ? (
              <p className="px-2 text-xs text-muted-foreground">
                No guilds. Invite the bot to a server to manage it here.
              </p>
            ) : (
              <select
                value={guildId}
                onChange={(e) => navigate(`/dashboard/guild/${e.target.value}`)}
                className="h-8 w-full rounded-none border border-input bg-transparent px-2 text-xs outline-none focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/50"
              >
                {guilds.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <nav className="space-y-0.5">
            <p className="mb-1 px-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {current ? current.name : "This server"}
            </p>
            {links.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-2 rounded-none border-l-2 px-2 py-1.5 text-sm transition-colors",
                    isActive
                      ? "border-primary bg-primary/10 font-medium text-foreground"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  )
                }
              >
                <Icon className="size-4" />
                {label}
              </NavLink>
            ))}

            <div className="my-3 border-t border-border" />
            <p className="mb-1 px-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Your app
            </p>
            <NavLink
              to="/dashboard"
              end
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-none border-l-2 px-2 py-1.5 text-sm transition-colors",
                  isActive
                    ? "border-primary bg-primary/10 font-medium text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                )
              }
            >
              <LayoutDashboard className="size-4" />
              Guilds
            </NavLink>
            <Link
              to="/dashboard/profile"
              className="flex items-center gap-2 rounded-none border-l-2 border-transparent px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              <User className="size-4" />
              Profile
            </Link>
            <Link
              to="/dashboard/diary"
              className="flex items-center gap-2 rounded-none border-l-2 border-transparent px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              <BookOpen className="size-4" />
              Diary
            </Link>
          </nav>
        </aside>

        <main className="min-w-0 flex-1">
          {/* Mobile: the server's pages as a horizontal tab strip instead of the sidebar */}
          <nav className="mb-4 flex overflow-x-auto border-b border-border text-sm lg:hidden">
            {links.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    "flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2 text-xs transition-colors",
                    isActive
                      ? "border-primary font-medium text-foreground"
                      : "border-transparent text-muted-foreground"
                  )
                }
              >
                <Icon className="size-3.5" />
                {label}
              </NavLink>
            ))}
          </nav>
          <Outlet key={guildId} />
        </main>
      </div>
    </div>
  )
}