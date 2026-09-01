# Actuation flow

```text
immutable input root + meta .gooo + typed candidate + fixed corpus
                           |
                    parse and bind candidate
                    /                  \
          forbidden effect          incomplete target
              REFUTED                  UNKNOWN
                    |
       copy input root to caller-owned temp
                    |
             apply candidate in temp
                    |
      lower -> generate Go -> execute Go
                    |
          compare stage N with stage N+1
                    |
              CLOSED or REFUTED
```

The original input root is only read and snapshotted. A valid candidate gets a
new output directory and a `stage-n-plus-1` artifact set; no branch performs a
repository operation or promotes the result.
