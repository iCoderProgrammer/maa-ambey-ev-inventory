package access

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

const accessKey = "access.resolved"

// Handler serves the access endpoints and the permission guard.
type Handler struct {
	loader        Loader
	allowProposed bool
	log           *slog.Logger
}

// NewHandler builds the HTTP adapter. allowProposed comes from
// RBAC_ALLOW_PROPOSED_GRANTS (config refuses it in staging/production).
func NewHandler(loader Loader, allowProposed bool, log *slog.Logger) *Handler {
	return &Handler{loader: loader, allowProposed: allowProposed, log: log}
}

// Register mounts the routes. They sit behind the session middleware (B-06).
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/me", h.Me)
	rg.GET("/access/roles", h.Require(PermRolesView), h.Roles)
}

// Current returns the caller's resolved access, loading it once per request.
// Use it in handlers/services for object-level checks (Has, Scope.Check).
func (h *Handler) Current(c *gin.Context) (Access, bool) {
	if v, ok := c.Get(accessKey); ok {
		a, ok := v.(Access)
		return a, ok
	}
	p, ok := auth.CurrentPrincipal(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "Please sign in to continue.")
		return Access{}, false
	}
	a, err := h.loader.LoadAccess(c.Request.Context(), p.UserID, h.allowProposed)
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "access lookup failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.Error(c, http.StatusServiceUnavailable, httpx.CodeUnavailable,
			"Your permissions could not be checked right now. Try again shortly.")
		return Access{}, false
	}
	c.Set(accessKey, a)
	return a, true
}

// Require denies the request with 403 unless the caller holds permission.
// It panics at start-up for a key missing from the catalogue, so a typo can
// never turn into a check that silently passes or always fails.
func (h *Handler) Require(permission string) gin.HandlerFunc {
	return h.RequireAny(permission)
}

// RequireAny denies the request with 403 unless the caller holds at least one
// of the permissions (e.g. selling-price or cost-price view). Services still
// check which one applies to the object. Unknown keys panic at start-up.
func (h *Handler) RequireAny(permissions ...string) gin.HandlerFunc {
	if len(permissions) == 0 {
		panic("access: RequireAny needs at least one permission")
	}
	for _, p := range permissions {
		if !slices.Contains(AllPermissions, p) {
			panic(fmt.Sprintf("access: unknown permission %q", p))
		}
	}
	return func(c *gin.Context) {
		a, ok := h.Current(c)
		if !ok {
			return
		}
		if !slices.ContainsFunc(permissions, a.Has) {
			h.log.WarnContext(c.Request.Context(), "permission denied",
				slog.String("request_id", c.GetString(httpx.RequestIDKey)),
				slog.String("user_id", a.UserID), slog.String("permission", strings.Join(permissions, "|")),
				slog.String("route", c.Request.Method+" "+c.FullPath()))
			Forbidden(c)
			return
		}
		c.Next()
	}
}

// Forbidden writes the standard 403. Do not say which permission was missing.
func Forbidden(c *gin.Context) {
	httpx.Error(c, http.StatusForbidden, httpx.CodeForbidden, "You do not have permission to do this.")
}

// MeUser identifies the caller.
type MeUser struct {
	ID          string `json:"id" example:"5b0f8a52-3c1e-4d6f-9a8b-0c1d2e3f4a5b"`
	Username    string `json:"username" example:"asha.verma"`
	DisplayName string `json:"displayName" example:"Asha Verma"`
} //@name MeUser

// LocationScope is the caller's location scope.
type LocationScope struct {
	All       bool       `json:"all" example:"false"`
	Locations []Location `json:"locations"`
} //@name LocationScope

// MeResponse is the permissions contract the UI renders from.
type MeResponse struct {
	User          MeUser        `json:"user"`
	Roles         []Role        `json:"roles"`
	Permissions   []string      `json:"permissions" example:"access.roles.view"`
	LocationScope LocationScope `json:"locationScope"`
	// True only when unapproved (proposed) grants are switched on (local development).
	ProposedGrantsActive bool `json:"proposedGrantsActive" example:"false"`
} //@name Me

// Me godoc
//
//	@Summary		Current user's roles, permissions and location scope
//	@Description	The permissions contract for the signed-in user. Permissions come only from owner-approved grants (plus proposed grants when RBAC_ALLOW_PROPOSED_GRANTS is on in local development). Empty until the permission matrix is approved. The UI uses this to decide what to show; every endpoint checks again.
//	@Tags			access
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=access.MeResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		503	{object}	httpx.ErrorResponse
//	@Router			/me [get]
func (h *Handler) Me(c *gin.Context) {
	p, _ := auth.CurrentPrincipal(c)
	a, ok := h.Current(c)
	if !ok {
		return
	}
	roles := a.Roles
	if roles == nil {
		roles = []Role{}
	}
	locations := a.Scope.Locations
	if locations == nil {
		locations = []Location{}
	}
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, MeResponse{
		User:                 MeUser{ID: p.UserID, Username: p.Username, DisplayName: p.DisplayName},
		Roles:                roles,
		Permissions:          a.Permissions(),
		LocationScope:        LocationScope{All: a.Scope.All, Locations: locations},
		ProposedGrantsActive: a.ProposedGrantsActive,
	})
}

// Roles godoc
//
//	@Summary		Role and permission matrix
//	@Description	Every role, its grants (PROPOSED or APPROVED with approver and reference) and the permission catalogue. matrixStatus is PENDING_OWNER_SIGN_OFF while no grant is approved. Read-only; grants change only through the audited cmd/access tool. Requires access.roles.view.
//	@Tags			access
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=access.Matrix}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		503	{object}	httpx.ErrorResponse
//	@Router			/access/roles [get]
func (h *Handler) Roles(c *gin.Context) {
	m, err := h.loader.RoleMatrix(c.Request.Context())
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "role matrix lookup failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
		return
	}
	httpx.OK(c, m)
}
