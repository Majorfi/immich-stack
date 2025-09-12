# What’s New: Advanced Filtering & Safer Grouping

This branch introduces easier, more powerful ways to tell Immich Stack how to group your photos into stacks — with simple, copy‑pasteable examples. It focuses on new capabilities; no test/refactor details here.

## Highlights

- Expression‑based criteria: combine rules with AND / OR / NOT
- Flexible groups: per‑group AND/OR logic (independent OR criteria)
- Safer regex: no match = not grouped by that rule (prevents stray stacks)
- CLI flag precedence: `--criteria` overrides `CRITERIA` env
- Better logs: clearer mode banners and per‑asset context when debugging

## 1) Expression‑Based Criteria (AND / OR / NOT)

You can now describe complex logic for grouping in one place. Expressions let you nest conditions and mix operators.

Example: group photos if the filename starts with PXL or IMG, and were taken within 2 seconds of each other.

```json
{
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
              "regex": { "key": "^PXL_", "index": 0 }
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "^IMG_", "index": 0 }
            }
          }
        ]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 2000 }
        }
      }
    ]
  }
}
```

Why it’s useful:

- Mix camera types and time windows in one rule
- Exclude things with `NOT` (e.g., not archived)
- Build exactly the grouping logic you want

## 2) Flexible Groups (Independent OR Criteria)

Prefer simpler building blocks? You can still use “groups” with an operator for that group. This is great for “match any of these patterns” cases.

Example: group by directory OR by being close in time.

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "OR",
      "criteria": [
        { "key": "originalPath", "split": { "delimiters": ["/"], "index": 2 } },
        { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
      ]
    }
  ]
}
```

Why it’s useful:

- Keep rules readable while still allowing OR logic
- Works well for “either same folder OR taken close in time”

## 3) Safer Regex Matching (No Surprise Grouping)

When a regex doesn’t match, that rule contributes no value. Assets won’t get grouped by an unmatched pattern. This prevents accidental stacking from partial or wrong matches.

Tip: Use `index` to grab exactly the part you want (full match = 0; capture group = 1+).

```json
{ "key": "originalFileName", "regex": { "key": "PXL_(\\d{8})", "index": 1 } }
```

## 4) CLI > Env: Explicit Criteria Wins

The `--criteria` flag now wins over the `CRITERIA` environment variable. This makes quick experiments easy.

Examples:

- One‑off run with CLI flag:

  ```bash
  immich-stack \
    --api-url "http://immich:3001/api" \
    --api-key "$API_KEY" \
    --criteria '{"mode":"advanced","groups":[{"operator":"OR","criteria":[{"key":"originalPath","split":{"delimiters":["/"],"index":2}},{"key":"localDateTime","delta":{"milliseconds":1000}}]}]}'
  ```

- Using `.env`, but override on the command line:

  ```bash
  CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","regex":{"key":"^IMG_","index":0}}},{"criteria":{"key":"localDateTime","delta":{"milliseconds":1500}}}]}}' \
  immich-stack --criteria '{"mode":"advanced","expression":{"operator":"OR","children":[{"criteria":{"key":"originalFileName","regex":{"key":"^PXL_","index":0}}},{"criteria":{"key":"originalFileName","regex":{"key":"^DSC","index":0}}}]}}'
  ```

## 5) Debugging Is Clearer

Turn on debug logging to see how assets are grouped and why.

```bash
LOG_LEVEL=debug immich-stack --api-url "$API_URL" --api-key "$API_KEY" --criteria '<your json here>'
```

What you’ll notice:

- A banner indicating which mode you’re using (expression vs groups)
- Per‑asset logs with filenames, IDs, and timestamps
- Clear “parent” vs “child” lines for each stack

## Quick Copy/Paste Recipes

- Group by base filename before a `~` or `.`:

  ```json
  [
    {
      "key": "originalFileName",
      "split": { "delimiters": ["~", "."], "index": 0 }
    }
  ]
  ```

- Group PXL photos taken within 1 second:

  ```json
  {
    "mode": "advanced",
    "expression": {
      "operator": "AND",
      "children": [
        {
          "criteria": {
            "key": "originalFileName",
            "regex": { "key": "^PXL_", "index": 0 }
          }
        },
        {
          "criteria": {
            "key": "localDateTime",
            "delta": { "milliseconds": 1000 }
          }
        }
      ]
    }
  }
  ```

- Group by folder name OR within 2 seconds:

  ```json
  {
    "mode": "advanced",
    "groups": [
      {
        "operator": "OR",
        "criteria": [
          {
            "key": "originalPath",
            "split": { "delimiters": ["/"], "index": 2 }
          },
          { "key": "localDateTime", "delta": { "milliseconds": 2000 } }
        ]
      }
    ]
  }
  ```

- One‑liner `.env` example (paste into your terminal for a single run):

  ```bash
  CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}}]' immich-stack --api-url "$API_URL" --api-key "$API_KEY"
  ```

## Backward Compatibility

- Legacy array format still works as before
- Advanced mode is opt‑in (`{"mode":"advanced": ...}`)
- If you don’t set `mode`, legacy behavior applies

That’s it — more control, safer grouping, and easier runs from the CLI.
