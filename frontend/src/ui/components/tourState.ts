const KEY = 'ares-bib-logger:tour-seen'
export const TOUR_EVENT = 'ares-bib-logger:tour-open'
/** Fired by the tour so the Admin page opens its Setup section before a step highlights something inside it. */
export const OPEN_SETUP_EVENT = 'ares-bib-logger:open-setup'

export interface TourStep {
  title: string
  body: string
  /** Page to show while this step is up. */
  route: string
  /** Elements to highlight, tried in order: the first one on the page wins. */
  targets: string[]
  /** Open the Admin Setup section first. */
  opensSetup?: boolean
}

/** After the welcome screen: what a new user needs to know, in the order they will meet it. */
export const TOUR_STEPS: TourStep[] = [
  {
    title: 'Set up in Admin',
    route: '/admin',
    targets: ['.setup-guide'],
    body: 'The setup guide at the top of Admin lists the steps in the order an event is best set up, and tells you what is left to do. Each step has a Go button.',
  },
  {
    title: 'Create or pick an event',
    route: '/admin',
    targets: ['#setup-event'],
    opensSetup: true,
    body: 'Create an event here, or select one as the active event. You can also import an event config exported from another station.',
  },
  {
    title: 'Races and checkpoints',
    route: '/admin',
    targets: ['#setup-races', '.setup-guide'],
    opensSetup: true,
    body: 'Add each race, then its checkpoints in order. Set distances in miles from the start if you want arrival projections. Lock the checkpoint order once it is right.',
  },
  {
    title: 'Roster',
    route: '/admin',
    targets: ['#setup-roster', '.setup-guide'],
    opensSetup: true,
    body: 'Paste each race roster as bib, first name and last name. Importing locks the roster, so check it first. Late entries can be added later in Edit Runners.',
  },
  {
    title: 'Your active checkpoint',
    route: '/admin',
    targets: ['#setup-active', '#setup-races', '.setup-guide'],
    opensSetup: true,
    body: 'Pick the checkpoint your station is physically covering for each race. Every bib you log is recorded there.',
  },
  {
    title: 'Log bibs on race day',
    route: '/data-entry',
    targets: ['#tour-log-bib', '.MuiAlert-root'],
    body: 'Type the bib and press Enter. The cards above show who is still to come, who is through, and who is expected next. DNS, DNF and transfers are next to it.',
  },
  {
    title: 'Share times by radio',
    route: '/winlink-export',
    targets: ['#tab-winlink-export'],
    body: 'Winlink Export builds a time column for your checkpoint to send to other stations. Winlink Import fills in the times they send you.',
  },
]

/** True if the tour has never been shown here. Without working storage we can't remember it, so it stays quiet. */
export function tourUnseen(): boolean {
  try {
    return window.localStorage?.getItem(KEY) === null
  } catch {
    return false
  }
}

export function markTourSeen() {
  try {
    window.localStorage?.setItem(KEY, '1')
  } catch {
    // storage unavailable: nothing to remember it with
  }
}

export const openTour = () => window.dispatchEvent(new Event(TOUR_EVENT))
