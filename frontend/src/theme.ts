import { createTheme } from '@mui/material/styles'

export type ColorMode = 'dark' | 'light'

// ── Design tokens ─────────────────────────────────────────────────────────────
// Mirrors the Sweep Tracker app's theme tokens so the two apps look like siblings.
const tokens = {
  light: {
    bg: '#e9ebef',
    panel: '#ffffff',
    border: '#e1e4e9',
    text: '#14171f',
    text2: '#4a5160',
    text3: '#5b6272',
    selected: '#eef2ff',
    accent: '#2f5bea',
    ok: '#17794a',
    okBg: '#e3f6ea',
    warn: '#8a5a00',
    warnBg: '#fff1d6',
    bad: '#a3122f',
    badBg: '#fde6ea',
    neutralBg: '#eef0f4',
    shadow: '0 2px 12px rgba(20, 23, 31, .12)',
  },
  dark: {
    bg: '#0b0e12',
    panel: '#12161c',
    border: '#262c36',
    text: '#e8ecf1',
    text2: '#aab2c0',
    text3: '#9aa3b2',
    selected: '#1a2233',
    accent: '#2f5bea',
    ok: '#5fd69a',
    okBg: '#12301f',
    warn: '#f2c15b',
    warnBg: '#3a2c0c',
    bad: '#ff8fa3',
    badBg: '#3a1620',
    neutralBg: '#222833',
    shadow: '0 2px 12px rgba(0, 0, 0, .5)',
  },
}

const FONT = '"DM Sans Variable", system-ui, -apple-system, "Segoe UI", sans-serif'

// ── Theme factory ─────────────────────────────────────────────────────────────
export function createAppTheme(mode: ColorMode) {
  const t = tokens[mode]
  const dark = mode === 'dark'

  return createTheme({
    palette: {
      mode,
      primary: { main: t.accent, contrastText: '#fff' },
      success: { main: t.ok },
      warning: { main: t.warn },
      error: { main: t.bad },
      background: { default: t.bg, paper: t.panel },
      text: { primary: t.text, secondary: t.text2, disabled: t.text3 },
      divider: t.border,
      action: {
        hover: t.selected,
        selected: t.selected,
      },
    },

    typography: {
      fontFamily: FONT,
      h1: { fontSize: '3rem', fontWeight: 600, lineHeight: 1.2 },
      h2: { fontSize: '2.25rem', fontWeight: 600, lineHeight: 1.2 },
      h3: { fontSize: '1.875rem', fontWeight: 600, lineHeight: 1.2 },
      h4: { fontSize: '1.5rem', fontWeight: 600, lineHeight: 1.3 },
      h5: { fontSize: '1.5rem', fontWeight: 600, letterSpacing: '-0.01em', lineHeight: 1.3 },
      h6: { fontSize: '1.1rem', fontWeight: 600, lineHeight: 1.4 },
      subtitle1: { fontSize: '0.9375rem', fontWeight: 600 },
      subtitle2: { fontSize: '0.9375rem', fontWeight: 700 },
      body1: { fontSize: '0.9375rem' },
      body2: { fontSize: '0.875rem' },
      caption: { fontSize: '0.75rem' },
      button: { fontSize: '0.9375rem', fontWeight: 600, textTransform: 'none' },
    },

    shape: { borderRadius: 10 },

    components: {
      // ── Paper / Card: soft, rounded floating panels ───────────────────────
      MuiPaper: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${t.border}`,
            borderRadius: 18,
            boxShadow: t.shadow,
          },
        },
      },

      // ── AppBar: a rounded card floating above the page ────────────────────
      MuiAppBar: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            backgroundColor: t.panel,
            border: `1px solid ${t.border}`,
            borderRadius: 18,
            boxShadow: t.shadow,
            color: t.text,
          },
        },
      },

      // ── Buttons ───────────────────────────────────────────────────────────
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          root: { borderRadius: 10, padding: '6px 18px', minHeight: 40, fontWeight: 600 },
          contained: { '&:hover': { filter: 'brightness(1.08)', backgroundColor: t.accent } },
          outlined: {
            borderColor: t.border,
            color: t.text,
            backgroundColor: t.panel,
            '&:hover': { borderColor: t.text3, backgroundColor: t.selected },
          },
          text: { '&:hover': { backgroundColor: t.selected } },
        },
      },

      MuiIconButton: {
        styleOverrides: {
          root: {
            borderRadius: 12,
            '&:hover': { backgroundColor: t.selected },
          },
        },
      },

      // ── Inputs ────────────────────────────────────────────────────────────
      MuiOutlinedInput: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            backgroundColor: t.panel,
            '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: t.text3 },
          },
          notchedOutline: { borderColor: t.border },
        },
      },

      MuiInputLabel: {
        styleOverrides: { root: { color: t.text2 } },
      },

      // ── Chips: small square-ish badges with tinted fills ──────────────────
      MuiChip: {
        styleOverrides: {
          root: { borderRadius: 6, fontWeight: 600, fontSize: '0.75rem' },
          colorSuccess: { backgroundColor: t.okBg, color: t.ok },
          colorWarning: { backgroundColor: t.warnBg, color: t.warn },
          colorError: { backgroundColor: t.badBg, color: t.bad },
          colorDefault: { backgroundColor: t.neutralBg, color: t.text2 },
        },
      },

      // ── Alerts: tinted fills, no border ───────────────────────────────────
      MuiAlert: {
        styleOverrides: { root: { borderRadius: 14 } },
        variants: (
          [
            ['success', t.okBg, t.ok],
            ['warning', t.warnBg, t.warn],
            ['error', t.badBg, t.bad],
            ['info', t.selected, t.text],
          ] as const
        ).map(([severity, bg, fg]) => ({
          props: { severity, variant: 'standard' as const },
          style: {
            backgroundColor: bg,
            color: fg,
            '& .MuiAlert-icon': { color: severity === 'info' ? t.accent : fg },
          },
        })),
      },

      // ── Dialog / Drawer ───────────────────────────────────────────────────
      MuiDialog: {
        styleOverrides: {
          paper: { border: `1px solid ${t.border}`, backgroundImage: 'none', borderRadius: 18 },
        },
      },
      MuiDrawer: {
        styleOverrides: {
          paper: { backgroundImage: 'none', borderRadius: 0, boxShadow: t.shadow },
        },
      },

      // ── Tables ────────────────────────────────────────────────────────────
      MuiTableHead: {
        styleOverrides: {
          root: {
            '& .MuiTableCell-root': {
              backgroundColor: t.neutralBg,
              fontWeight: 600,
              fontSize: '0.75rem',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: t.text2,
            },
          },
        },
      },
      MuiTableRow: {
        styleOverrides: {
          root: {
            '&:hover': { backgroundColor: t.selected },
            '&:last-child td': { borderBottom: 0 },
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: { borderBottom: `1px solid ${t.border}`, padding: '8px 12px' },
        },
      },

      MuiDivider: {
        styleOverrides: { root: { borderColor: t.border } },
      },

      // ── Tabs: pills instead of an underline ───────────────────────────────
      MuiTabs: {
        styleOverrides: {
          root: { minHeight: 48, padding: '0 8px 8px' },
          indicator: { display: 'none' },
        },
      },
      MuiTab: {
        styleOverrides: {
          root: {
            fontWeight: 500,
            fontSize: '0.9375rem',
            textTransform: 'none',
            minHeight: 40,
            padding: '8px 14px',
            borderRadius: 10,
            color: t.text2,
            '&:hover': { backgroundColor: t.selected },
            '&.Mui-selected': {
              fontWeight: 600,
              color: t.text,
              backgroundColor: t.selected,
            },
          },
        },
      },

      // ── CssBaseline: global resets ────────────────────────────────────────
      MuiCssBaseline: {
        styleOverrides: {
          body: {
            backgroundColor: t.bg,
            scrollbarColor: `${dark ? '#3a4150' : '#c4c9d4'} transparent`,
          },
        },
      },
    },
  })
}
