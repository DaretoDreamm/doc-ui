package models

import "time"

type Project struct {
	ID           int64
	Name         string
	BaseURL      string
	Template     string
	ColorPalette string
	Typography   string
	Status       string
	PageCount    int
	ErrorMsg     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Page struct {
	ID        int64
	ProjectID int64
	URL       string
	Path      string
	Title     string
	Content   string
	NavOrder  int
	Depth     int
	CreatedAt time.Time
}

type CrawlProgress struct {
	ProjectID  int64
	Status     string
	PageCount  int
	CurrentURL string
	ErrorMsg   string
}

type TemplateOption struct {
	ID          string
	Name        string
	Description string
}

type ColorPaletteOption struct {
	ID    string
	Name  string
	Color string // display swatch color
}

type TypographyOption struct {
	ID      string
	Name    string
	Display string // display/heading font
	Body    string // body text font
	Code    string // code font
	Hint    string // CSS generic family for form preview
}

var Typographies = []TypographyOption{
	{ID: "modern", Name: "Modern", Display: "Syne", Body: "Plus Jakarta Sans", Code: "Fira Code", Hint: "sans-serif"},
	{ID: "classic", Name: "Classic", Display: "Playfair Display", Body: "Source Serif 4", Code: "IBM Plex Mono", Hint: "serif"},
	{ID: "geometric", Name: "Geometric", Display: "Outfit", Body: "DM Sans", Code: "JetBrains Mono", Hint: "sans-serif"},
	{ID: "mono", Name: "Mono", Display: "Space Mono", Body: "Space Mono", Code: "Space Mono", Hint: "monospace"},
	{ID: "humanist", Name: "Humanist", Display: "Fraunces", Body: "Nunito Sans", Code: "Fira Code", Hint: "serif"},
}

func GetTypographyFontsLink(id string) string {
	links := map[string]string{
		"modern":    `<link href="https://fonts.googleapis.com/css2?family=Syne:wght@400;500;600;700;800&family=Plus+Jakarta+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400&family=Fira+Code:wght@400;500;600&display=swap" rel="stylesheet"/>`,
		"classic":   `<link href="https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,400;0,500;0,600;0,700;0,800;1,400&family=Source+Serif+4:ital,wght@0,400;0,500;0,600;0,700;1,400&family=IBM+Plex+Mono:wght@400;500;600&display=swap" rel="stylesheet"/>`,
		"geometric": `<link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=DM+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet"/>`,
		"mono":      `<link href="https://fonts.googleapis.com/css2?family=Space+Mono:ital,wght@0,400;0,700;1,400&display=swap" rel="stylesheet"/>`,
		"humanist":  `<link href="https://fonts.googleapis.com/css2?family=Fraunces:ital,wght@0,400;0,500;0,600;0,700;0,800;1,400&family=Nunito+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400&family=Fira+Code:wght@400;500;600&display=swap" rel="stylesheet"/>`,
	}
	if l, ok := links[id]; ok {
		return l
	}
	return links["modern"]
}

func GetTypographyCSS(id string) string {
	css := map[string]string{
		"modern":    `:root{--font-display:'Syne',sans-serif;--font-body:'Plus Jakarta Sans',sans-serif;--font-code:'Fira Code',monospace;}`,
		"classic":   `:root{--font-display:'Playfair Display',serif;--font-body:'Source Serif 4',serif;--font-code:'IBM Plex Mono',monospace;}`,
		"geometric": `:root{--font-display:'Outfit',sans-serif;--font-body:'DM Sans',sans-serif;--font-code:'JetBrains Mono',monospace;}`,
		"mono":      `:root{--font-display:'Space Mono',monospace;--font-body:'Space Mono',monospace;--font-code:'Space Mono',monospace;}`,
		"humanist":  `:root{--font-display:'Fraunces',serif;--font-body:'Nunito Sans',sans-serif;--font-code:'Fira Code',monospace;}`,
	}
	if c, ok := css[id]; ok {
		return c
	}
	return css["modern"]
}

var Templates = []TemplateOption{
	{ID: "minimal", Name: "Minimal", Description: "Clean single-column reader"},
	{ID: "sidebar", Name: "Sidebar", Description: "Left navigation with content"},
	{ID: "modern", Name: "Modern", Description: "Multi-column with TOC"},
}

var ColorPalettes = []ColorPaletteOption{
	{ID: "noir", Name: "Noir", Color: "#18181b"},
	{ID: "snow", Name: "Snow", Color: "#f4f4f5"},
	{ID: "ocean", Name: "Ocean", Color: "#0ea5e9"},
	{ID: "forest", Name: "Forest", Color: "#16a34a"},
	{ID: "sunset", Name: "Sunset", Color: "#ea580c"},
	{ID: "grape", Name: "Grape", Color: "#7c3aed"},
	{ID: "rose", Name: "Rose", Color: "#e11d48"},
	{ID: "amber", Name: "Amber", Color: "#d97706"},
	{ID: "teal", Name: "Teal", Color: "#0d9488"},
	{ID: "slate", Name: "Slate", Color: "#64748b"},
}

// Each palette has both light and dark mode CSS.
// The modern template toggles .dark class on <html>.
func GetPaletteCSS(id string) string {
	p := paletteCSS[id]
	if p == "" {
		return paletteCSS["ocean"]
	}
	return p
}

var paletteCSS = map[string]string{
	"noir": `
		:root {
			--c-primary:#a1a1aa;--c-primary-hover:#d4d4d8;--c-bg:#09090b;--c-surface:#09090b;--c-sidebar:#0a0a0c;
			--c-text:#e4e4e7;--c-text-muted:#a1a1aa;--c-border:#27272a;--c-code-bg:#18181b;--c-code-text:#e4e4e7;
			--c-accent:#d4d4d8;--c-active-bg:#27272a;--c-active-text:#fafafa;--c-blockquote:#3f3f46;--c-link:#a1a1aa;
		}
		.dark {
			--c-primary:#a1a1aa;--c-primary-hover:#d4d4d8;--c-bg:#09090b;--c-surface:#09090b;--c-sidebar:#0a0a0c;
			--c-text:#e4e4e7;--c-text-muted:#a1a1aa;--c-border:#27272a;--c-code-bg:#18181b;--c-code-text:#e4e4e7;
			--c-accent:#d4d4d8;--c-active-bg:#27272a;--c-active-text:#fafafa;--c-blockquote:#3f3f46;--c-link:#a1a1aa;
		}
	`,
	"snow": `
		:root {
			--c-primary:#52525b;--c-primary-hover:#3f3f46;--c-bg:#ffffff;--c-surface:#ffffff;--c-sidebar:#fafafa;
			--c-text:#18181b;--c-text-muted:#52525b;--c-border:#e4e4e7;--c-code-bg:#fafafa;--c-code-text:#18181b;
			--c-accent:#71717a;--c-active-bg:#f4f4f5;--c-active-text:#18181b;--c-blockquote:#d4d4d8;--c-link:#3f3f46;
		}
		.dark {
			--c-primary:#a1a1aa;--c-primary-hover:#d4d4d8;--c-bg:#18181b;--c-surface:#1f1f23;--c-sidebar:#18181b;
			--c-text:#e4e4e7;--c-text-muted:#a1a1aa;--c-border:#3f3f46;--c-code-bg:#27272a;--c-code-text:#e4e4e7;
			--c-accent:#d4d4d8;--c-active-bg:#27272a;--c-active-text:#fafafa;--c-blockquote:#52525b;--c-link:#d4d4d8;
		}
	`,
	"ocean": `
		:root {
			--c-primary:#0ea5e9;--c-primary-hover:#0284c7;--c-bg:#f8fafc;--c-surface:#ffffff;--c-sidebar:#f0f9ff;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#e2e8f0;--c-code-bg:#0c4a6e;--c-code-text:#e0f2fe;
			--c-accent:#38bdf8;--c-active-bg:#e0f2fe;--c-active-text:#0369a1;--c-blockquote:#0ea5e9;--c-link:#0284c7;
		}
		.dark {
			--c-primary:#38bdf8;--c-primary-hover:#7dd3fc;--c-bg:#0c1222;--c-surface:#111a2e;--c-sidebar:#0c1222;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#1e3a5f;--c-code-bg:#0c4a6e;--c-code-text:#e0f2fe;
			--c-accent:#7dd3fc;--c-active-bg:#0c4a6e;--c-active-text:#bae6fd;--c-blockquote:#0369a1;--c-link:#38bdf8;
		}
	`,
	"forest": `
		:root {
			--c-primary:#16a34a;--c-primary-hover:#15803d;--c-bg:#f8fdf8;--c-surface:#ffffff;--c-sidebar:#f0fdf4;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#dcfce7;--c-code-bg:#14532d;--c-code-text:#dcfce7;
			--c-accent:#4ade80;--c-active-bg:#dcfce7;--c-active-text:#15803d;--c-blockquote:#16a34a;--c-link:#15803d;
		}
		.dark {
			--c-primary:#4ade80;--c-primary-hover:#86efac;--c-bg:#0a1210;--c-surface:#0f1a16;--c-sidebar:#0a1210;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#14532d;--c-code-bg:#14532d;--c-code-text:#dcfce7;
			--c-accent:#86efac;--c-active-bg:#14532d;--c-active-text:#bbf7d0;--c-blockquote:#15803d;--c-link:#4ade80;
		}
	`,
	"sunset": `
		:root {
			--c-primary:#ea580c;--c-primary-hover:#c2410c;--c-bg:#faf8f5;--c-surface:#ffffff;--c-sidebar:#fff7ed;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#fed7aa;--c-code-bg:#7c2d12;--c-code-text:#ffedd5;
			--c-accent:#fb923c;--c-active-bg:#ffedd5;--c-active-text:#c2410c;--c-blockquote:#ea580c;--c-link:#c2410c;
		}
		.dark {
			--c-primary:#fb923c;--c-primary-hover:#fdba74;--c-bg:#120c08;--c-surface:#1a1008;--c-sidebar:#120c08;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#7c2d12;--c-code-bg:#7c2d12;--c-code-text:#ffedd5;
			--c-accent:#fdba74;--c-active-bg:#7c2d12;--c-active-text:#fed7aa;--c-blockquote:#c2410c;--c-link:#fb923c;
		}
	`,
	"grape": `
		:root {
			--c-primary:#7c3aed;--c-primary-hover:#6d28d9;--c-bg:#faf8ff;--c-surface:#ffffff;--c-sidebar:#faf5ff;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#ddd6fe;--c-code-bg:#2e1065;--c-code-text:#ede9fe;
			--c-accent:#a78bfa;--c-active-bg:#ede9fe;--c-active-text:#6d28d9;--c-blockquote:#7c3aed;--c-link:#6d28d9;
		}
		.dark {
			--c-primary:#a78bfa;--c-primary-hover:#c4b5fd;--c-bg:#0e0a18;--c-surface:#150f24;--c-sidebar:#0e0a18;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#2e1065;--c-code-bg:#2e1065;--c-code-text:#ede9fe;
			--c-accent:#c4b5fd;--c-active-bg:#2e1065;--c-active-text:#ddd6fe;--c-blockquote:#6d28d9;--c-link:#a78bfa;
		}
	`,
	"rose": `
		:root {
			--c-primary:#e11d48;--c-primary-hover:#be123c;--c-bg:#fdf2f4;--c-surface:#ffffff;--c-sidebar:#fff1f2;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#fecdd3;--c-code-bg:#4c0519;--c-code-text:#ffe4e6;
			--c-accent:#fb7185;--c-active-bg:#ffe4e6;--c-active-text:#be123c;--c-blockquote:#e11d48;--c-link:#be123c;
		}
		.dark {
			--c-primary:#fb7185;--c-primary-hover:#fda4af;--c-bg:#120608;--c-surface:#1a0a0e;--c-sidebar:#120608;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#4c0519;--c-code-bg:#4c0519;--c-code-text:#ffe4e6;
			--c-accent:#fda4af;--c-active-bg:#4c0519;--c-active-text:#fecdd3;--c-blockquote:#be123c;--c-link:#fb7185;
		}
	`,
	"amber": `
		:root {
			--c-primary:#d97706;--c-primary-hover:#b45309;--c-bg:#fffbeb;--c-surface:#ffffff;--c-sidebar:#fef3c7;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#fde68a;--c-code-bg:#78350f;--c-code-text:#fef3c7;
			--c-accent:#fbbf24;--c-active-bg:#fef3c7;--c-active-text:#b45309;--c-blockquote:#d97706;--c-link:#b45309;
		}
		.dark {
			--c-primary:#fbbf24;--c-primary-hover:#fcd34d;--c-bg:#120e04;--c-surface:#1a1406;--c-sidebar:#120e04;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#78350f;--c-code-bg:#78350f;--c-code-text:#fef3c7;
			--c-accent:#fcd34d;--c-active-bg:#78350f;--c-active-text:#fde68a;--c-blockquote:#b45309;--c-link:#fbbf24;
		}
	`,
	"teal": `
		:root {
			--c-primary:#0d9488;--c-primary-hover:#0f766e;--c-bg:#f0fdfa;--c-surface:#ffffff;--c-sidebar:#f0fdfa;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#99f6e4;--c-code-bg:#134e4a;--c-code-text:#ccfbf1;
			--c-accent:#2dd4bf;--c-active-bg:#ccfbf1;--c-active-text:#0f766e;--c-blockquote:#0d9488;--c-link:#0f766e;
		}
		.dark {
			--c-primary:#2dd4bf;--c-primary-hover:#5eead4;--c-bg:#0a1210;--c-surface:#0f1a17;--c-sidebar:#0a1210;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#134e4a;--c-code-bg:#134e4a;--c-code-text:#ccfbf1;
			--c-accent:#5eead4;--c-active-bg:#134e4a;--c-active-text:#99f6e4;--c-blockquote:#0f766e;--c-link:#2dd4bf;
		}
	`,
	"slate": `
		:root {
			--c-primary:#475569;--c-primary-hover:#334155;--c-bg:#f8fafc;--c-surface:#ffffff;--c-sidebar:#f1f5f9;
			--c-text:#0f172a;--c-text-muted:#64748b;--c-border:#e2e8f0;--c-code-bg:#1e293b;--c-code-text:#e2e8f0;
			--c-accent:#94a3b8;--c-active-bg:#e2e8f0;--c-active-text:#334155;--c-blockquote:#475569;--c-link:#334155;
		}
		.dark {
			--c-primary:#94a3b8;--c-primary-hover:#cbd5e1;--c-bg:#0f172a;--c-surface:#1e293b;--c-sidebar:#0f172a;
			--c-text:#e2e8f0;--c-text-muted:#94a3b8;--c-border:#334155;--c-code-bg:#1e293b;--c-code-text:#e2e8f0;
			--c-accent:#cbd5e1;--c-active-bg:#334155;--c-active-text:#e2e8f0;--c-blockquote:#475569;--c-link:#94a3b8;
		}
	`,
}
