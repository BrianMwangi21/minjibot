package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kibetnathan/minjibot/internal/domain/deletedmessage"
	"github.com/kibetnathan/minjibot/internal/domain/guild"
	"github.com/kibetnathan/minjibot/internal/ports/repository"
	authsvc "github.com/kibetnathan/minjibot/internal/services/auth"
	"github.com/labstack/echo/v5"
)

// fakeGuildRepo satisfies GuildRepository by embedding the interface (all other
// methods are nil and would panic if called, but the tested path only calls List).
type fakeGuildRepo struct {
	repository.GuildRepository
	guilds []guild.Guild
}

func (f *fakeGuildRepo) List(context.Context) ([]guild.Guild, error) { return f.guilds, nil }

type fakeDeletedRepo struct {
	repository.DeletedMessageRepository
}

func (f *fakeDeletedRepo) CountForAllGuilds(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (f *fakeDeletedRepo) ListForGuild(context.Context, string, int32, int32) ([]deletedmessage.DeletedMessage, error) {
	return nil, nil
}

func (f *fakeDeletedRepo) CountForGuild(context.Context, string) (int64, error) {
	return 0, nil
}

type fakeAuditRepo struct {
	repository.AuditLogRepository
}

func (f *fakeAuditRepo) CountForAllGuilds(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// newTestLogHandlers builds logHandlers wired to a session manager with a known
// secret and a per-guild authorizer: a session carrying "admin-token" holds
// moderation permissions in g1, any other token is a plain member of g1.
func newTestLogHandlers() (*logHandlers, *authsvc.SessionManager) {
	sess := authsvc.NewSessionManager("test-secret")
	return &logHandlers{
		sess: sess,
		authz: &guildAuthz{
			cache: make(map[string]guildPermEntry),
			fetchGuilds: func(_ context.Context, token string) ([]authsvc.Guild, error) {
				if token == "admin-token" {
					return []authsvc.Guild{{ID: "g1", Name: "Guild One", Permissions: discordgo.PermissionManageGuild}}, nil
				}
				return []authsvc.Guild{{ID: "g1", Name: "Guild One", Permissions: discordgo.PermissionSendMessages}}, nil
			},
		},
		guilds:  &fakeGuildRepo{guilds: []guild.Guild{{ID: "g1", Name: "Guild One"}}},
		audits:  &fakeAuditRepo{},
		deletes: &fakeDeletedRepo{},
	}, sess
}

// doListGuilds runs listGuilds with an optional session cookie for userID
// (empty = no cookie) and returns the recorded response.
func doListGuilds(t *testing.T, h *logHandlers, sess *authsvc.SessionManager, userID, token string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/guilds", nil)
	if userID != "" {
		cookie, err := sess.Create(userID, token)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h.listGuilds(c); err != nil {
		t.Fatalf("listGuilds returned error: %v", err)
	}
	return rec
}

// doListDeletedMessages runs listDeletedMessages with the given query params and
// an optional session cookie.
func doListDeletedMessages(t *testing.T, h *logHandlers, sess *authsvc.SessionManager, userID, token, guildID string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/logs/deleted?guild_id="+guildID, nil)
	if userID != "" {
		cookie, err := sess.Create(userID, token)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h.listDeletedMessages(c); err != nil {
		t.Fatalf("listDeletedMessages returned error: %v", err)
	}
	return rec
}

func TestListGuilds_NoSession_401(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListGuilds(t, h, sess, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without a session, got %d", rec.Code)
	}
}

func TestListGuilds_Moderator_200(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListGuilds(t, h, sess, "admin-id", "admin-token")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for a moderator session, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Guild One") {
		t.Errorf("expected authorized guild to be listed, got body: %s", rec.Body.String())
	}
}

func TestListGuilds_PlainMember_EmptyList(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListGuilds(t, h, sess, "member-id", "member-token")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (empty list) for a plain member, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("expected an empty list for a plain member, got body: %s", rec.Body.String())
	}
}

func TestListGuilds_WrongSecret_401(t *testing.T) {
	h, _ := newTestLogHandlers()
	other := authsvc.NewSessionManager("a-different-secret")
	rec := doListGuilds(t, h, other, "admin-id", "admin-token")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for a session signed with the wrong secret, got %d", rec.Code)
	}
}

func TestListDeletedMessages_NoSession_401(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListDeletedMessages(t, h, sess, "", "", "g1")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without a session, got %d", rec.Code)
	}
}

func TestListDeletedMessages_NotModerator_403(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListDeletedMessages(t, h, sess, "member-id", "member-token", "g1")
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for a non-moderator, got %d", rec.Code)
	}
}

func TestListDeletedMessages_Moderator_200(t *testing.T) {
	h, sess := newTestLogHandlers()
	rec := doListDeletedMessages(t, h, sess, "admin-id", "admin-token", "g1")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for a moderator, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
