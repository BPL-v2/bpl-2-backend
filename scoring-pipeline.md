# Scoring Pipeline

This document explains how the scoring process works end-to-end, from collecting player data to delivering scores to the frontend.

---

## Overview

```
GGG API (items + characters)
    ↓
Filter to event players → Kafka queue
    ↓
Matching → ObjectiveMatch records in DB
    ↓
Aggregation → one Match per team/objective
    ↓
Evaluation → points + ranks via ScoringRules
    ↓
Diff vs. cached state → ScoreDiff
    ↓
WebSocket push + REST API
```

---

## Stage 1: Data Collection

Two streams of data are continuously collected during an event.

### Character Data

Every few minutes the system polls the Path of Exile API for each registered player's character: level, class, ascendancy, atlas progress, and equipment. Equipment changes are queued for deeper stat analysis via Path of Building, which provides derived stats like DPS, health, resistances, and block chance.

### Item Data

Simultaneously the system watches over different item sources:

- the public stash API — a live feed from GGG that announces whenever any player moves items in their stash. This feed is filtered down to only items belonging to players registered for the current event, in the right league.
- the guild stash API - a current snapshot of a guilds stash tabs. Team leads can configure, which stash tabs will be polled.

Every Stash Change is published to a **Kafka topic** — a durable, ordered message queue. This creates a complete log of every item change for the event that can be replayed if needed.

---

## Stage 2: Matching

A separate process consumes the Kafka queue and checks each item change against all active **objectives** (the challenges in the event).

Each objective has a set of **conditions** — rules such as:

- The item must be corrupted
- The base type must be "Forbidden Flesh"
- The item level must be above 84
- The explicit modifiers must contain a specific mod

A list of Conditions is combined with AND logic. The system evaluates every item against every objective's condition tree and records any hit.

When a match is found, an **ObjectiveMatch** record is written to the database. This is the atomic unit of the whole system. It captures:

- Which objective was matched
- Which team and player (if it can be determined) achieved it
- How many (e.g. stack size or count)
- The exact timestamp

Manual **submissions** — where a player uploads proof of a completion — go through a review and approval flow. Once approved they are converted into the same ObjectiveMatch format and flow through the same downstream pipeline.

---

## Stage 3: Aggregation

Raw matches are noisy — a single team might have hundreds of ObjectiveMatch rows for the same objective (every time the item was moved, every stack update). The aggregation step collapses these into **one canonical result per team per objective**, using a method defined per objective:

| Method                   | What it does                                                                       |
| ------------------------ | ---------------------------------------------------------------------------------- |
| **FirstCompletion**      | Whichever team reached the required amount earliest                                |
| **FirstFreshCompletion** | Same, but only counts items if they are still present in the latest stash movement |
| **LatestValue**          | The most recent value recorded                                                     |
| **HighestValue**         | The peak value achieved                                                            |
| **LowestValue**          | The lowest value (useful for things like fastest kill time)                        |
| **ValueChangeInWindow**  | The difference between a start and end timestamp                                   |

The result is a single **Match** per team per objective: an aggregated number (count or value), a timestamp, and whether the team has "finished" the objective (hit the required amount).

---

## Stage 4: Evaluation (Scoring)

The aggregated Match objects are fed through **scoring rules**. Each objective can have one or more ScoringRules that define how points are awarded. There are nine rule types:

| Rule                          | How points work                                                                                   |
| ----------------------------- | ------------------------------------------------------------------------------------------------- |
| **FixedPointsOnCompletion**   | Everyone who finishes gets the same flat points                                                   |
| **RankByCompletionTime**      | Teams ranked by how fast they finished; first place gets the most                                 |
| **RankByHighestValue**        | Ranked by who achieved the highest number                                                         |
| **RankByLowestValue**         | Ranked by lowest value (e.g. fastest time)                                                        |
| **PointsByValue**             | Points scale continuously with the value achieved using a curve formula                           |
| **RankByChildCompletionTime** | Ranked by how fast a team completed N sub-objectives                                              |
| **BonusPerChildCompletion**   | A parent objective gives bonus points for each sub-objective completed                            |
| **BingoBoardRanking**         | A grid of objectives — ranked by how many bingo lines (rows, columns, diagonals) a team completes |
| **RankByChildValueSum**       | Ranked by the sum of values across all sub-objectives                                             |

The output is a **Score** per team per objective: points, rank, whether it is finished, timestamp, and the contributing player.

The total points for a team on an objective is the sum across all its ScoringRule results, plus any bonus points awarded by parent objectives.

---

## Stage 5: Score Distribution

Scores are recalculated whenever there are active viewers or an API request comes in. The new result is compared to the last cached state, producing a **ScoreDiff** — only the things that actually changed.

Diffs are delivered in two ways:

- **WebSocket stream** — The frontend keeps a live connection open and receives score updates every ~5 seconds. There is a detailed version (full per-objective breakdown) and a simple version (just total points per team).
- **REST API** — For one-time fetches of the current score state.
