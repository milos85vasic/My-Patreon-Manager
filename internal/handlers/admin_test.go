package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminReload_PackageFunction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/reload", nil)
	AdminReload(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"config_reloaded"}`, w.Body.String())
}

func TestAdminSyncStatus_PackageFunction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/sync/status", nil)
	AdminSyncStatus(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"idle","active_sync":false}`, w.Body.String())
}

func TestRegisterAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterAdminRoutes(r, slog.Default())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/reload", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/admin/sync/status", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestAdminHandler_Reload_EmitsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(slog.Default())
	require.NotNil(t, h.AuditStore())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/reload", nil)
	h.Reload(c)

	assert.Equal(t, http.StatusOK, w.Code)
	entries, err := h.AuditStore().List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "admin", entries[0].Actor)
	assert.Equal(t, "admin.reload", entries[0].Action)
	assert.Equal(t, "config", entries[0].Target)
	assert.Equal(t, "ok", entries[0].Outcome)
	assert.False(t, entries[0].CreatedAt.IsZero())
}

func TestAdminHandler_SyncStatus_NoAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(slog.Default())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/sync/status", nil)
	h.SyncStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	entries, _ := h.AuditStore().List(context.Background(), 10)
	assert.Len(t, entries, 0)
}

func TestAdminHandler_SetAuditStore(t *testing.T) {
	h := NewAdminHandler(slog.Default())
	custom := audit.NewRingStore(8)
	h.SetAuditStore(custom)
	assert.Same(t, custom, h.AuditStore())

	// Resetting with nil falls back to a fresh ring store, never nil.
	h.SetAuditStore(nil)
	assert.NotNil(t, h.AuditStore())
	assert.NotSame(t, custom, h.AuditStore())
}

func TestAdminHandler_EmitAudit_StampsCreatedAt(t *testing.T) {
	h := NewAdminHandler(slog.Default())
	h.emitAudit(context.Background(), audit.Entry{
		Actor:  "admin",
		Action: "admin.test",
	})
	entries, err := h.AuditStore().List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].CreatedAt.IsZero())
}
