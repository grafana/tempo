// This file implements the prepare-downscale HTTP contract and the shutdown-
// marker bookkeeping behind it -- the 2026-07-16 shutdown-semantics redesign
// (DESIGN.md § Availability model amendment). Companion to
// bloomgateway_ring.go's KeepInstanceInTheRingOnShutdown wiring: that file
// makes an ordinary graceful stop keep this instance ACTIVE in the ring by
// default -- BasicLifecycler.stopping() reads that flag DIRECTLY, no
// delegate involved (bloomgateway_ring.go's own doc comment explains why a
// stopping delegate was tried and removed) -- this file is the operator's
// explicit, persistent "no, I actually mean to remove this instance" signal
// that flips the SAME flag back to the pre-redesign unregister-on-stop
// behavior for exactly one stop.
//
// Modeled on modules/livestore/downscale.go's OWN PrepareDownscaleHandler --
// and ONLY that handler. Live-store's OTHER downscale handler
// (PreparePartitionDownscaleHandler) is Kafka-partition-ring machinery
// (ring.PartitionInstanceLifecycler, states PartitionPending/Active/Inactive)
// with no analogue here: bloom-gateway has no partition ring at all (every
// instance consumes every Kafka partition independently, DESIGN.md §
// Consumers). Notably, live-store's OWN read/token ring (a plain
// ring.BasicLifecycler, the same type this package uses) has no keep-in-ring
// conditioning anywhere -- its PrepareDownscaleHandler only ever toggles
// partition-ring settings. This file is therefore new territory for Tempo,
// not a copy-paste of existing token-ring wiring; only the shape (marker
// file, GET/POST/DELETE, checked first in starting()) and the underlying
// pkg/util/shutdownmarker package are actually shared with live-store.
package bloomgateway

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/shutdownmarker"
)

// PrepareDownscaleHandler prepares this instance for an intentional,
// one-time removal from the ring -- the operator's explicit override of
// this package's new "keep in ring on a graceful stop" default (§
// Availability model amendment; bloomgateway_ring.go's
// KeepInstanceInTheRingOnShutdown wiring). Contract mirrors modules/
// livestore/downscale.go's own PrepareDownscaleHandler exactly (see this
// file's own package doc comment for what does and doesn't transfer from
// live-store):
//
//   - GET: reports whether prepare-for-downscale is currently set ("set\n"
//     or "unset\n", matching live-store's own plain-text response).
//   - POST: creates the shutdown marker (pkg/util/shutdownmarker; survives
//     a restart landing between this call and the operator's actual stop
//     -- see checkShutdownMarker) and flips this instance's lifecycler to
//     unregister-on-stop. The NEXT graceful stop unregisters directly:
//     BasicLifecycler.stopping() reads the flag and calls
//     unregisterInstance itself, with no LEAVING stopover -- there is no
//     way to reach LEAVING from a delegate outside the ring package once
//     stopping() has begun (bloomgateway_ring.go's own doc comment), and
//     none is needed: this package's ringOp={ACTIVE} treats LEAVING and
//     absent identically, so the missing stopover changes nothing
//     observable. Net effect is the same as every stop before this
//     redesign: the instance leaves the ring.
//   - DELETE: reverses both -- removes the marker and restores
//     keep-in-ring-on-stop, so a subsequent graceful stop goes back to
//     costing survivors nothing.
//
// Guarded on g.State() == services.Running, matching live-store's own
// guard: flipping this mid-start or mid-stop is a caller error, not a
// state this handler should try to reconcile.
func (g *BloomGateway) PrepareDownscaleHandler(w http.ResponseWriter, r *http.Request) {
	if g.State() != services.Running {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	markerPath := shutdownmarker.GetPath(g.cfg.ShutdownMarkerDir)
	switch r.Method {
	case http.MethodGet:
		exists, err := shutdownmarker.Exists(markerPath)
		if err != nil {
			level.Error(g.logger).Log("msg", "bloomgateway: unable to check for prepare-downscale marker", "path", markerPath, "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists {
			util.WriteTextResponse(w, "set\n")
		} else {
			util.WriteTextResponse(w, "unset\n")
		}
		return

	case http.MethodPost:
		if err := shutdownmarker.Create(markerPath); err != nil {
			level.Error(g.logger).Log("msg", "bloomgateway: unable to create prepare-downscale marker", "path", markerPath, "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		g.setPrepareDownscale()
		level.Info(g.logger).Log("msg", "bloomgateway: prepared for downscale; the next graceful stop will unregister this instance from the ring", "path", markerPath)

	case http.MethodDelete:
		if err := shutdownmarker.Remove(markerPath); err != nil {
			level.Error(g.logger).Log("msg", "bloomgateway: unable to remove prepare-downscale marker", "path", markerPath, "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		g.unsetPrepareDownscale()
		level.Info(g.logger).Log("msg", "bloomgateway: prepare-downscale cancelled; graceful stops keep this instance in the ring again", "path", markerPath)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setPrepareDownscale/unsetPrepareDownscale are the two places that flip
// this instance's lifecycler between "keep in ring" and "unregister" on
// its NEXT graceful stop -- called from PrepareDownscaleHandler (an
// explicit POST/DELETE) and from checkShutdownMarker (re-arming across a
// restart that lands between a POST and the operator's actual stop).
// SetKeepInstanceInTheRingOnShutdown is safe to call before the lifecycler
// service itself has started (New() already constructs g.ringManager.
// Lifecycler; this only flips an atomic field on it, vendor/.../dskit/ring/
// basic_lifecycler.go) -- checkShutdownMarker relies on exactly that,
// calling this before the ring subservices are even started.
func (g *BloomGateway) setPrepareDownscale() {
	g.ringManager.Lifecycler.SetKeepInstanceInTheRingOnShutdown(false)
}

func (g *BloomGateway) unsetPrepareDownscale() {
	g.ringManager.Lifecycler.SetKeepInstanceInTheRingOnShutdown(true)
}

// checkShutdownMarker re-arms prepare-downscale (setPrepareDownscale) if
// this instance was already marked for it before this process started --
// see PrepareDownscaleHandler's own doc comment for the full contract, and
// starting()'s own call site for why this must run before the ring
// subservices do. Creates cfg.ShutdownMarkerDir if missing, mirroring
// live-store's own starting()-time precedent (modules/livestore/
// live_store.go: "this needs to be done as first thing because... it may
// change the behaviour of startup").
func (g *BloomGateway) checkShutdownMarker() error {
	if _, err := os.Stat(g.cfg.ShutdownMarkerDir); os.IsNotExist(err) {
		if err := os.MkdirAll(g.cfg.ShutdownMarkerDir, 0o700); err != nil {
			return fmt.Errorf("creating shutdown marker directory: %w", err)
		}
	}

	markerPath := shutdownmarker.GetPath(g.cfg.ShutdownMarkerDir)
	exists, err := shutdownmarker.Exists(markerPath)
	if err != nil {
		return fmt.Errorf("checking shutdown marker: %w", err)
	}
	if exists {
		level.Info(g.logger).Log("msg", "bloomgateway: detected existing prepare-downscale marker; the next graceful stop will unregister this instance from the ring", "path", markerPath)
		g.setPrepareDownscale()
	}
	return nil
}
