# Geo

This package is the backend geographic lookup layer.

Current scope:

- sovereign airspace ownership lookup for the Iran-war theater
- coarse territorial-water lookup for the Iran-war theater
- path sampling for transit / strike validation

This package is the first step in replacing the older country-only helper logic
that previously lived in `internal/sim/theater_countries.go`.

Planned expansion:

- separate land ownership from airspace ownership
- add richer maritime zones beyond territorial waters
- return structured violation contexts for frontend explainability

Do not add scenario or diplomacy logic here. This package should only answer
geographic ownership / context questions.
