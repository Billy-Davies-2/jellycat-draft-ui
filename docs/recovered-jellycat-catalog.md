# Recovered Jellycat Catalog

The recovered catalog from the pre-refresh admin page is stored in `data/recovered-jellycats.json`.

It contains 24 entries, including intentional duplicate Jellycats. The position codes were recovered from the card badges:

- `CH`
- `CC`
- `HH`
- `SS`

To restore production while signed in as an admin:

1. Open `https://jellycat.daviestechlabs.io/admin` in the browser session that is signed in as admin.
2. Open the browser DevTools console.
3. Run the contents of `scripts/restore-recovered-jellycats.js`.
4. Confirm the prompt. The script deletes the current undrafted catalog first, then creates the 24 recovered Jellycats.

The script stops before deleting anything if any current player is already drafted.