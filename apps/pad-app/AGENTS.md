## Flutter Pad Group

This subtree is owned by the Flutter Pad team.

### Owned scope
- `lib/`
- `test/`
- `pubspec.yaml`
- `pubspec.lock`
- `macos/`
- `web/`

### Language
- Dart / Flutter only

### Goal
- Deliver a stable child task board client across Chrome, tablet, and mobile targets

### Do not do
- Do not modify Go backend source
- Do not modify Parent Web source
- Do not redefine backend API semantics from the client side

### Default policy
- Treat backend responses as contracts
- If an API change is needed, request it from the Go-API group instead of patching backend code yourself
- Focus on loading, empty, error, and sync states as first-class UX
