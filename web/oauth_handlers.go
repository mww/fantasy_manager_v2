package web

import (
	"net/http"
	"time"

	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/model"
	"github.com/unrolled/render"
)

func oauthLinkHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := ctrl.OAuthStart(model.PlatformYahoo)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

func oauthRedirectHandler(ctrl controller.C, render *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		code := params.Get("code")
		state := params.Get("state")

		err := ctrl.OAuthExchange(r.Context(), state, code)
		if err != nil {
			render.HTML(w, http.StatusInternalServerError, "500", err)
			return
		}

		data := map[string]any{
			"state": state,
			"year":  time.Now().Format("2006"),
		}
		render.HTML(w, http.StatusOK, "addYahooLeague", data)
	}
}
