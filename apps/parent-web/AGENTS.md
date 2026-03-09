## Parent Web Group

This subtree is owned by the Parent Web team.

### Owned scope
- `src/`
- `package.json`
- `package-lock.json`
- `index.html`

### Language
- JavaScript / React only

### Goal
- Deliver a stable parent workflow for input, parse preview, review, and confirm

### Do not do
- Do not modify Go backend source
- Do not modify Flutter Pad source
- Do not redefine parser field meanings in the UI

### Default policy
- Use backend parser fields as the source of truth
- Optimize for fast parent review of risky tasks
- If response contracts need to change, hand the request to the Go groups
