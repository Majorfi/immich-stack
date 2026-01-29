# Real-World Examples

Common stacking scenarios and how to solve them with immich-auto-stack.

## Defaults at a glance

These examples assume the defaults unless overridden:

- **Criteria**: `originalFileName` split on `["~", "."]` (index `0`) + `localDateTime` within `1000ms`
- **Parent filename promote**: `cover,edit,crop,hdr,biggestNumber`
- **Parent ext promote**: `.jpg,.png,.jpeg,.heic,.dng`

**Parent selection order**:

1. Regex `promote_index` (if present)
2. Parent filename promote (order matters)
3. `biggestNumber` (only when in the promote list)
4. Parent ext promote (order matters)
5. Extension rank (`jpeg > jpg > png > others`) when not explicitly promoted
6. Alphabetical (case-sensitive)

**Notes**:

- Order matters in both promote lists.
- `biggestNumber` only works on numeric suffixes **after delimiters found in your `originalFileName` split**.  
  If you want `-1` / `_2` to count, add `-` or `_` to `split.delimiters`.

## Quick verification

Run in dry-run + debug to see grouping and parent selection in logs:

```sh
LOG_LEVEL=debug immich-stack --dry-run
```

All grouping/parent-selection scenarios below have matching tests in `pkg/stacker/examples_test.go`.

## RAW + JPEG Pairing

### Canon / Nikon / Sony (same filename, different extension)

**Problem:** Your camera produces `IMG_1234.jpg` and `IMG_1234.CR2` (or `.NEF`, `.ARW`). You want them grouped as a single stack with the JPEG on top.

**Solution:** The default configuration handles this out of the box. No custom criteria needed.

```sh
API_KEY=your_key
API_URL=http://immich-server:2283/api
```

The default criteria splits on `~` and `.` to extract the base filename (`IMG_1234`) and groups assets taken within 1 second of each other. The default `PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.dng` ensures common processed formats win over RAW.

### Fujifilm RAF + JPEG

**Problem:** Your Fujifilm camera produces `DSCF1234.jpg` and `DSCF1234.RAF`. Same as above but with `.RAF` extension.

**Solution:** Default configuration works. `.jpg` is promoted by default, so the JPEG will be the stack parent.

### Samsung Galaxy (JPG + DNG)

**Problem:** Samsung phones produce `20240115_143022.jpg` and `20240115_143022.dng` when shooting RAW+JPEG.

**Solution:** Default configuration works. Both `.jpg` and `.dng` are in the default extension promote list, with `.jpg` having higher priority.

### Apple iPhone ProRAW (HEIC + DNG)

**Problem:** iPhones with ProRAW enabled produce `IMG_1234.HEIC` and `IMG_1234.DNG`. You want them stacked with the HEIC on top.

**Solution:** Default configuration works. Both `.heic` and `.dng` are in the default extension promote list, with `.heic` having higher priority.

## Google Pixel Photos

### Pixel RAW + JPEG (standard)

**Problem:** Pixel phones use a specific naming pattern:

- `PXL_20260121_195958829.RAW-01.COVER.jpg`
- `PXL_20260121_195958829.RAW-02.ORIGINAL.dng`

The default split on `["~", "."]` produces `PXL_20260121_195958829` from both files (index 0 after splitting on all delimiters). This works with default config out of the box.

**Solution:** Default configuration works. For explicit control:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.dng
```

### Pixel Motion Photos

**Problem:** Pixel creates a Motion Photo variant alongside the regular photo:

- `PXL_20240115_143022345.jpg`
- `PXL_20240115_143022345.MP.jpg`

You want them stacked together.

**Solution:** Default configuration groups them (both resolve to `PXL_20240115_143022345` after splitting on `["~", "."]`). To control which file is on top:

To put the **Motion Photo on top** (default behavior): no change needed. The case-sensitive alphabetical tiebreaker sorts `.MP.jpg` before `.jpg` (uppercase `M` < lowercase `j`), so the Motion Photo is the parent by default.

To put the **regular photo on top**:

```sh
PARENT_FILENAME_PROMOTE=,mp,cover,edit,crop,hdr,biggestNumber
```

The leading empty string promotes files that do **not** contain `mp` (or any other keyword), making the plain `PXL_*.jpg` the parent.

### Pixel 10 Pro Triple Grouping (30x Zoom)

**Problem:** The Pixel 10 Pro creates three files for high-zoom shots:

- `PXL_20260120_120000000.jpg` (original JPEG)
- `PXL_20260120_120000000.dng` (RAW)
- `PXL_20260120_120000000.NIGHT.jpg` (AI-processed)

All three should be in one stack with the original JPEG on top.

**Solution:** Default configuration groups them (the split on `["~", "."]` extracts the same base filename from all three). However, the two `.jpg` files tie on extension promotion, and the case-sensitive alphabetical tiebreaker puts `.NIGHT.jpg` before `.jpg` (uppercase `N` < lowercase `j`).

To ensure the original JPEG is on top, use negative matching:

```sh
PARENT_FILENAME_PROMOTE=,night,cover,edit,crop,hdr,biggestNumber
```

The leading empty string promotes files that do **not** contain `night` (or any other keyword), making the plain `.jpg` the parent.

## Google Photos Edited Versions

**Problem:** Google Photos exports include edited copies alongside originals:

- `vacation_sunset.jpg`
- `vacation_sunset-edited.jpg`

You want them stacked together.

**Solution:** Add `-` to the split delimiters so both files resolve to the same base name:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
PARENT_FILENAME_PROMOTE=edit,cover,crop,hdr,biggestNumber
```

The split on `["-", "~", "."]` extracts `vacation_sunset` from both files. `PARENT_FILENAME_PROMOTE` with `edit` first makes the edited version the parent since its filename contains "edited". To put the **original on top** instead, use negative matching:

```sh
PARENT_FILENAME_PROMOTE=,edit,cover,crop,hdr,biggestNumber
```

The leading empty string promotes files that do **not** contain `edit` (or any other keyword), making the plain `vacation_sunset.jpg` the parent. Simply removing `edit` from the promote list is not enough — the alphabetical tiebreaker would still pick `vacation_sunset-edited.jpg` because `-` sorts before `.`.

## RAW+JPEG with Lightroom Numeric Edits

**Problem:** You shoot RAW+JPEG and use Lightroom to export edited versions with numeric suffixes. Your library contains:

- `ABC001.ARW`
- `ABC001.JPEG`
- `ABC001-1.JPEG`
- `ABC001-2.JPEG`

You want all four in one stack, with the latest edit (`ABC001-2.JPEG`) as the thumbnail.

**Solution:** Add `-` to the split delimiters so all files resolve to the same base name `ABC001`:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.dng,.arw
```

The default `PARENT_FILENAME_PROMOTE` already includes `biggestNumber`, which picks `ABC001-2.JPEG` (highest numeric suffix) as the parent. `PARENT_EXT_PROMOTE` with processed formats first ensures a JPG/PNG wins over the RAW when no numeric edit exists.

This also works with the Pixel RAW naming pattern (`PXL_*.RAW-01.COVER.jpg` / `PXL_*.RAW-02.ORIGINAL.dng`) since the same split logic extracts the shared base. The default promote list includes `cover`, so the COVER JPEG can be selected as parent.

## Photoshop Workflows

### RAW + JPEG + PSD (Photoshop Project Files)

**Problem:** You shoot RAW+JPEG and edit in Photoshop, keeping the `.psd` project file alongside exports. Your library contains:

- `IMG_1234.CR2` (RAW)
- `IMG_1234.jpg` (camera JPEG)
- `IMG_1234.psd` (Photoshop project)

You want all files grouped with the JPEG on top.

**Solution:** Default configuration groups files with the same base filename. The `.psd` extension is listed after JPEG formats in the default extension promote list, so JPEG files become the stack thumbnail:

```sh
API_KEY=your_key
API_URL=http://immich-server:2283/api
```

### RAW + JPEG + PSD with Final Export

**Problem:** You also export a final edited version alongside the source files:

- `IMG_1234.CR2` (RAW)
- `IMG_1234.jpg` (camera JPEG)
- `IMG_1234.psd` (Photoshop project)
- `IMG_1234-final.jpg` (exported edit)

You want all files grouped with the final export on top.

**Solution:** Add `-` to the split delimiters so `IMG_1234-final.jpg` groups with the others:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
PARENT_FILENAME_PROMOTE=final,edit,cover,crop,hdr,biggestNumber
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.psd,.dng,.cr2
```

The split on `["-", "~", "."]` extracts `IMG_1234` from all files. The `final` keyword promotes `IMG_1234-final.jpg` as parent.

### Photoshop with Versioned Exports

**Problem:** You use Photoshop's "Save As" to create multiple export versions:

- `portrait.psd` (project file)
- `portrait_1.jpg` (first export)
- `portrait_2.jpg` (second export)
- `portrait_final.jpg` (final version)

You want them stacked with the final version on top.

**Solution:** Add `_` to the split delimiters so all files resolve to the same base name:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["_","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":86400000}}]'
PARENT_FILENAME_PROMOTE=final,biggestNumber
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.psd
```

The split on `["_", "~", "."]` extracts `portrait` from all files. `final` in the promote list ensures `portrait_final.jpg` is the parent. The `biggestNumber` fallback handles versioned files (`_2` > `_1`) when no `final` file exists. The large time delta (24 hours) accommodates files created across editing sessions.

**Note:** `biggestNumber` requires pure numeric suffixes (e.g., `_1`, `_2`). Files named `_v1`, `_v2` will fall back to alphabetical ordering.

To put the **PSD on top** instead (useful if you primarily work in Photoshop):

```sh
PARENT_EXT_PROMOTE=.psd,.jpg,.png,.jpeg
```

### Aperture/Lightroom Vault with Photoshop Edits

**Problem:** You migrated from Aperture or Lightroom and have a mix of RAW, JPEG, and Photoshop files with various naming conventions:

- `IMG_1234.CR2` (original RAW)
- `IMG_1234.jpg` (original JPEG)
- `IMG_1234-Edit.psd` (Photoshop edit)
- `IMG_1234-Edit.jpg` (exported from Photoshop)

You want all four grouped with the exported edit on top.

**Solution:** Add `-` to the split delimiters:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":86400000}}]'
PARENT_FILENAME_PROMOTE=edit,cover,crop,hdr,biggestNumber
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.psd,.dng,.cr2
```

This puts `IMG_1234-Edit.jpg` on top (contains "edit" + has `.jpg` extension priority over `.psd`).

## Burst Photos

### Camera Bursts with Shared Timestamp

**Problem:** Some cameras embed a shared timestamp in burst filenames:

- `DSCPDC_0000_BURST20180828114700954.JPG`
- `DSCPDC_0001_BURST20180828114700954.JPG`
- `DSCPDC_0002_BURST20180828114700954.JPG`
- `DSCPDC_0003_BURST20180828114700954_COVER.JPG`

You want them grouped into one stack with the cover shot on top.

**Solution:** Use a regex to extract the shared BURST timestamp for grouping, and `sequence` for ordering:

```sh
CRITERIA='[{"key":"originalFileName","regex":{"key":"BURST(\\d+)","index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
PARENT_FILENAME_PROMOTE=cover,sequence
```

The regex extracts `20180828114700954` from all four files, grouping them. The `cover` keyword promotes the `_COVER` file as parent. The `sequence` keyword orders the rest numerically.

### Sequential Burst Photos with Common Prefix

**Problem:** Your camera names bursts as `photo_0001.jpg`, `photo_0002.jpg`, `photo_0003.jpg`. They share a common prefix but have different sequence numbers.

**Solution:** Use a regex to extract the common prefix for grouping, then `sequence` to control parent order:

```sh
CRITERIA='[{"key":"originalFileName","regex":{"key":"^(.+?)_\\d+\\.","index":1}},{"key":"localDateTime","delta":{"milliseconds":3000}}]'
PARENT_FILENAME_PROMOTE=sequence,cover,edit,crop,hdr
```

The regex extracts `photo` from all three files, grouping them together. The `sequence` keyword sorts the stack by the numeric portion, making the first in sequence the parent.

**Important:** The `sequence` keyword controls **parent selection order** within an already-grouped stack. It does not affect grouping itself. The regex criterion is what makes the files group together.

### Limitation: Fully Sequential Filenames

Photos with completely different base filenames (e.g., `IMG_1234.jpg`, `IMG_1235.jpg`, `IMG_1236.jpg`) cannot be reliably grouped by filename since no shared portion can be extracted. Apple iPhone bursts fall into this category as they rely on EXIF BurstUUID metadata, which is not available through the Immich API.

## Parent Selection Control

### Always Show Processed Files on Top (Lightroom Behavior)

**Problem:** You want processed files (JPEG, PNG, HEIC) always displayed as the stack representative, with RAW files accessible but hidden behind them.

**Solution:**

```sh
PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.dng,.cr2,.cr3,.nef,.arw,.raf,.orf,.rw2
```

List processed formats first. RAW formats at the end means they'll never be chosen as parent when a processed file exists.

### Always Show RAW on Top

**Problem:** You primarily work with RAW files and want them as the stack representative.

**Solution:**

```sh
PARENT_EXT_PROMOTE=.dng,.cr2,.cr3,.nef,.arw,.raf,.orf,.rw2,.jpg,.png,.jpeg
```

### Prefer Edited Files on Top

**Problem:** You have edited versions alongside originals and want the edit to always be the stack parent.

**Solution:**

```sh
PARENT_FILENAME_PROMOTE=final,edit,crop,hdr,cover,biggestNumber
```

Files containing "final" get highest priority, then "edit", etc.

## Mixed Camera Setups

### Multiple Cameras with Different Naming

**Problem:** You shoot with both a Pixel phone and a Canon DSLR. Pixel files are `PXL_*.jpg` + `PXL_*.dng`, Canon files are `IMG_*.JPG` + `IMG_*.CR2`. You want both sets to stack correctly.

**Solution:** Default configuration handles both since it groups by filename before extension + timestamp. Both naming patterns work with the default `split` on `"."`.

For explicit camera-aware grouping with an OR expression:

```sh
CRITERIA='{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "operator": "OR",
        "children": [
          {
            "criteria": {
              "key": "originalFileName",
              "regex": {"key": "^(PXL_\\d+_\\d+)", "index": 1}
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "regex": {"key": "^(IMG_\\d+)", "index": 1}
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "split": {"delimiters": ["~", "."], "index": 0}
            }
          }
        ]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": {"milliseconds": 1000}
        }
      }
    ]
  }
}'
```

The OR expression tries Pixel naming first, then Canon, then falls back to generic split. The AND with `localDateTime` ensures time proximity.

## Docker Compose Full Example

A complete `docker-compose.yml` for Pixel RAW+JPEG with edited file promotion:

```yaml
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    environment:
      - API_KEY=your_immich_api_key
      - API_URL=http://immich-server:2283/api
      - RUN_MODE=cron
      - CRON_INTERVAL=3600
      - PARENT_EXT_PROMOTE=.jpg,.png,.jpeg,.heic,.dng
      - PARENT_FILENAME_PROMOTE=cover,edit,crop,hdr,biggestNumber
      - LOG_LEVEL=info
    restart: unless-stopped
```
