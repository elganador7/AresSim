# Engineering Workflow

This folder contains repo-wide engineering guidance that should stay stable
across feature branches.

## Validation Tiers

### Fast

Use for normal feature work before and during a coding pass.

Runs:

- targeted backend unit packages
- frontend test suite
- frontend production build

Command:

```bash
./scripts/validate_fast.sh
```

### Full

Use before committing larger changes or when touching bridge-level / app-level
behavior.

Runs:

- everything in `fast`
- full Go test suite across the repo

Command:

```bash
./scripts/validate_full.sh
```

### Slow

Use when changing real-data ingest, caches, or proving-ground behavior.

Runs:

- oil render-cache rebuild
- proving-ground sweep

Command:

```bash
./scripts/validate_slow.sh
```

## Intended Usage

- default feature loop: `fast`
- pre-commit for larger backend changes: `full`
- data/model calibration changes: `slow`

The goal is to keep the default workflow fast while still preserving one clear
path for heavier validation.
