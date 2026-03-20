# Library

This folder loads the unit-definition library from YAML and converts it into DB records used by the editor and simulator.

- `loader.go`: recursive YAML loading, validation, normalization, and record conversion.
- `data/default/`: built-in library content shipped with the app.

The loader is the right place for backward-compatible defaults. Prefer normalizing older YAML here instead of scattering fallback logic across the app.

For the longer-term platform-library quality bar and classification plan, see [docs/platform_library/README.md](/Users/cameronspringer/AresSim/docs/platform_library/README.md).
