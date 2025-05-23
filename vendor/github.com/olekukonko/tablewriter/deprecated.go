package tablewriter

import "github.com/olekukonko/tablewriter/tw"

// Deprecated: WithBorders is no longer used.
// Border control has been moved to the renderer, which now manages its own borders.
// This Option has no effect on the Table and may be removed in future versions.
func WithBorders(borders tw.Border) Option {
	return func(target *Table) {
		if target.renderer != nil {
			cfg := target.renderer.Config()
			cfg.Borders = borders
			if target.logger != nil {
				target.logger.Debugf("Option: WithBorders applied to Table: %+v", borders)
			}
		}
	}
}

// Deprecated: WithBorders is no longer supported.
// Use [tw.Behavior] directly to configure border settings.
type Behavior tw.Behavior

// Deprecated: WithRendererSettings i sno longer supported.
type Settings tw.Settings

// WithRendererSettings updates the renderer's settings (e.g., separators, lines).
// Render setting has move to renders directly
// you can also use WithRendition for renders that have rendition support
func WithRendererSettings(settings tw.Settings) Option {
	return func(target *Table) {
		if target.renderer != nil {
			cfg := target.renderer.Config()
			cfg.Settings = settings
			if target.logger != nil {
				target.logger.Debugf("Option: WithRendererSettings applied to Table: %+v", settings)
			}
		}
	}
}
