import { createTheme, alpha } from '@mui/material/styles'

export type ColorMode = 'dark' | 'light'

// ── Color primitives ──────────────────────────────────────────────────────────
const brand = {
  50: 'hsl(210, 100%, 95%)',
  100: 'hsl(210, 100%, 92%)',
  200: 'hsl(210, 100%, 80%)',
  300: 'hsl(210, 100%, 65%)',
  400: 'hsl(210, 98%,  48%)',
  500: 'hsl(210, 98%,  42%)',
  600: 'hsl(210, 98%,  55%)',
  700: 'hsl(210, 100%, 35%)',
  800: 'hsl(210, 100%, 16%)',
  900: 'hsl(210, 100%, 21%)',
}

const gray = {
  50: 'hsl(220, 35%, 97%)',
  100: 'hsl(220, 30%, 94%)',
  200: 'hsl(220, 20%, 88%)',
  300: 'hsl(220, 20%, 80%)',
  400: 'hsl(220, 20%, 65%)',
  500: 'hsl(220, 20%, 42%)',
  600: 'hsl(220, 20%, 35%)',
  700: 'hsl(220, 20%, 25%)',
  800: 'hsl(220, 30%,  6%)',
  900: 'hsl(220, 35%,  3%)',
}

// ── Theme factory ─────────────────────────────────────────────────────────────
export function createAppTheme(mode: ColorMode) {
  const dark = mode === 'dark'

  return createTheme({
    palette: {
      mode,
      primary: {
        main: dark ? brand[600] : brand[500],
        light: brand[300],
        dark: brand[700],
        contrastText: dark ? gray[50] : '#fff',
      },
      background: {
        default: dark ? gray[900] : gray[50],
        paper: dark ? gray[800] : '#fff',
      },
      text: {
        primary: dark ? 'hsl(0, 0%, 100%)' : gray[900],
        secondary: dark ? gray[400] : gray[600],
      },
      divider: dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8),
      action: {
        hover: dark ? alpha(gray[600], 0.2) : alpha(gray[200], 0.7),
        selected: dark ? alpha(gray[600], 0.3) : alpha(gray[200], 0.9),
      },
    },

    typography: {
      fontFamily: '"Inter", system-ui, sans-serif',
      h1: { fontSize: '3rem', fontWeight: 600, lineHeight: 1.2 },
      h2: { fontSize: '2.25rem', fontWeight: 600, lineHeight: 1.2 },
      h3: { fontSize: '1.875rem', fontWeight: 600, lineHeight: 1.2 },
      h4: { fontSize: '1.5rem', fontWeight: 600, lineHeight: 1.3 },
      h5: { fontSize: '1.25rem', fontWeight: 600, lineHeight: 1.3 },
      h6: { fontSize: '1.125rem', fontWeight: 600, lineHeight: 1.4 },
      subtitle1: { fontSize: '0.875rem', fontWeight: 600 },
      subtitle2: { fontSize: '0.8125rem', fontWeight: 600 },
      body1: { fontSize: '0.875rem' },
      body2: { fontSize: '0.8125rem' },
      caption: { fontSize: '0.75rem' },
      button: { fontSize: '0.875rem', fontWeight: 600, textTransform: 'none' },
    },

    shape: { borderRadius: 8 },

    shadows: dark
      ? [
          'none',
          `0 1px 2px ${alpha(gray[900], 0.6)}`,
          `0 2px 4px ${alpha(gray[900], 0.5)}`,
          `0 4px 8px ${alpha(gray[900], 0.4)}`,
          `0 6px 12px ${alpha(gray[900], 0.4)}`,
          `0 8px 16px ${alpha(gray[900], 0.4)}`,
          `0 10px 20px ${alpha(gray[900], 0.3)}`,
          `0 12px 24px ${alpha(gray[900], 0.3)}`,
          `0 14px 28px ${alpha(gray[900], 0.3)}`,
          `0 16px 32px ${alpha(gray[900], 0.3)}`,
          `0 18px 36px ${alpha(gray[900], 0.3)}`,
          `0 20px 40px ${alpha(gray[900], 0.3)}`,
          `0 22px 44px ${alpha(gray[900], 0.3)}`,
          `0 24px 48px ${alpha(gray[900], 0.3)}`,
          `0 26px 52px ${alpha(gray[900], 0.3)}`,
          `0 28px 56px ${alpha(gray[900], 0.3)}`,
          `0 30px 60px ${alpha(gray[900], 0.3)}`,
          `0 32px 64px ${alpha(gray[900], 0.3)}`,
          `0 34px 68px ${alpha(gray[900], 0.3)}`,
          `0 36px 72px ${alpha(gray[900], 0.3)}`,
          `0 38px 76px ${alpha(gray[900], 0.3)}`,
          `0 40px 80px ${alpha(gray[900], 0.3)}`,
          `0 42px 84px ${alpha(gray[900], 0.3)}`,
          `0 44px 88px ${alpha(gray[900], 0.3)}`,
          `0 46px 92px ${alpha(gray[900], 0.3)}`,
        ]
      : [
          'none',
          `0 1px 2px ${alpha(gray[900], 0.08)}`,
          `0 2px 4px ${alpha(gray[900], 0.08)}`,
          `0 4px 8px ${alpha(gray[900], 0.07)}`,
          `0 6px 12px ${alpha(gray[900], 0.07)}`,
          `0 8px 16px ${alpha(gray[900], 0.06)}`,
          `0 10px 20px ${alpha(gray[900], 0.06)}`,
          `0 12px 24px ${alpha(gray[900], 0.05)}`,
          `0 14px 28px ${alpha(gray[900], 0.05)}`,
          `0 16px 32px ${alpha(gray[900], 0.05)}`,
          `0 18px 36px ${alpha(gray[900], 0.05)}`,
          `0 20px 40px ${alpha(gray[900], 0.05)}`,
          `0 22px 44px ${alpha(gray[900], 0.04)}`,
          `0 24px 48px ${alpha(gray[900], 0.04)}`,
          `0 26px 52px ${alpha(gray[900], 0.04)}`,
          `0 28px 56px ${alpha(gray[900], 0.04)}`,
          `0 30px 60px ${alpha(gray[900], 0.04)}`,
          `0 32px 64px ${alpha(gray[900], 0.04)}`,
          `0 34px 68px ${alpha(gray[900], 0.04)}`,
          `0 36px 72px ${alpha(gray[900], 0.04)}`,
          `0 38px 76px ${alpha(gray[900], 0.04)}`,
          `0 40px 80px ${alpha(gray[900], 0.04)}`,
          `0 42px 84px ${alpha(gray[900], 0.04)}`,
          `0 44px 88px ${alpha(gray[900], 0.04)}`,
          `0 46px 92px ${alpha(gray[900], 0.04)}`,
        ],

    components: {
      // ── Paper / Card ──────────────────────────────────────────────────────
      MuiPaper: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8)}`,
          },
        },
      },

      // ── AppBar ────────────────────────────────────────────────────────────
      MuiAppBar: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            backgroundColor: dark ? gray[900] : gray[100],
            borderBottom: `1px solid ${dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8)}`,
            color: dark ? 'hsl(0, 0%, 100%)' : gray[900],
          },
        },
      },

      // ── Buttons ───────────────────────────────────────────────────────────
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          root: {
            borderRadius: 8,
            padding: '6px 16px',
            fontWeight: 600,
          },
          contained: {
            background: dark ? brand[600] : brand[500],
            '&:hover': { background: brand[700] },
          },
          outlined: {
            borderColor: dark ? alpha(gray[500], 0.5) : alpha(gray[400], 0.6),
            '&:hover': {
              borderColor: dark ? gray[400] : gray[600],
              background: dark ? alpha(gray[600], 0.15) : alpha(gray[200], 0.5),
            },
          },
        },
      },

      MuiIconButton: {
        styleOverrides: {
          root: { borderRadius: 8 },
        },
      },

      // ── Inputs ────────────────────────────────────────────────────────────
      MuiOutlinedInput: {
        styleOverrides: {
          notchedOutline: {
            borderColor: dark ? alpha(gray[500], 0.35) : alpha(gray[400], 0.5),
          },
          root: {
            '&:hover .MuiOutlinedInput-notchedOutline': {
              borderColor: dark ? gray[400] : gray[500],
            },
          },
        },
      },

      MuiInputLabel: {
        styleOverrides: {
          root: { color: dark ? gray[400] : gray[600] },
        },
      },

      // ── Select ────────────────────────────────────────────────────────────
      MuiSelect: {
        styleOverrides: {
          root: {
            '& .MuiOutlinedInput-notchedOutline': {
              borderColor: dark ? alpha(gray[500], 0.35) : alpha(gray[400], 0.5),
            },
          },
        },
      },

      // ── Chips ─────────────────────────────────────────────────────────────
      MuiChip: {
        styleOverrides: {
          root: { borderRadius: 6, fontWeight: 500 },
        },
      },

      // ── Alerts ────────────────────────────────────────────────────────────
      MuiAlert: {
        styleOverrides: {
          root: { borderRadius: 8 },
        },
      },

      // ── Dialog ────────────────────────────────────────────────────────────
      MuiDialog: {
        styleOverrides: {
          paper: {
            border: `1px solid ${dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8)}`,
            backgroundImage: 'none',
          },
        },
      },

      // ── Tables ────────────────────────────────────────────────────────────
      MuiTableHead: {
        styleOverrides: {
          root: {
            '& .MuiTableCell-root': {
              backgroundColor: dark ? alpha(gray[700], 0.5) : gray[100],
              fontWeight: 600,
              fontSize: '0.75rem',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: dark ? gray[300] : gray[600],
            },
          },
        },
      },
      MuiTableRow: {
        styleOverrides: {
          root: {
            '&:hover': { backgroundColor: dark ? alpha(gray[700], 0.3) : alpha(gray[100], 0.8) },
            '&:last-child td': { borderBottom: 0 },
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: {
            borderBottom: `1px solid ${dark ? alpha(gray[600], 0.2) : alpha(gray[200], 0.9)}`,
            padding: '8px 12px',
          },
        },
      },

      // ── Divider ───────────────────────────────────────────────────────────
      MuiDivider: {
        styleOverrides: {
          root: { borderColor: dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8) },
        },
      },

      // ── Tabs ──────────────────────────────────────────────────────────────
      MuiTabs: {
        styleOverrides: {
          root: {
            borderBottom: `1px solid ${dark ? alpha(gray[600], 0.3) : alpha(gray[300], 0.8)}`,
          },
        },
      },
      MuiTab: {
        styleOverrides: {
          root: {
            fontWeight: 500,
            fontSize: '0.875rem',
            textTransform: 'none',
            minHeight: 40,
            padding: '8px 16px',
            '&.Mui-selected': { fontWeight: 600 },
          },
        },
      },

      // ── CssBaseline: global resets ────────────────────────────────────────
      MuiCssBaseline: {
        styleOverrides: {
          body: {
            backgroundColor: dark ? gray[900] : gray[50],
            scrollbarColor: `${dark ? gray[600] : gray[300]} transparent`,
            '&::-webkit-scrollbar': { width: 8 },
            '&::-webkit-scrollbar-thumb': {
              background: dark ? gray[600] : gray[300],
              borderRadius: 4,
            },
          },
        },
      },
    },
  })
}
