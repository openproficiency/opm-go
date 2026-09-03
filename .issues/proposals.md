# Model and schema public contract

The README identifies `model` and `schema` as public packages, but the linked
contract documents, `docs/model.md` and `docs/schema.md`, are absent. Those
documents should define the packages' responsibilities and their complete
public APIs before implementation exposes any declarations.

The model contract should specify how callers discover the targeted Open
Proficiency Model release, including the exact version representation and
whether compatibility information is also required. It should also resolve the
README's suggestion that `model` owns embedded schema copies while `schema`
owns bundled schemas, so schema ownership is unambiguous.

The schema contract should specify how callers identify and enumerate bundled
schemas, whether they can read immutable schema bytes or an `fs.FS`, and which
validation forms are supported. It should also define error behavior and
whether compiled schemas are part of the supported API. The existing
`internal/schema` package already embeds, compiles, and validates the bundled
schemas; an approved public API should delegate to that implementation rather
than duplicate it.

No model version or schema access declarations are proposed as final names or
signatures here because no documented contract currently selects them.

# Topic wire-form construction

The topic schema permits an inline topic object inside `subtopics`, while the
documented Go contract exposes `Topic.Subtopics` as `[]string`. The
implementation can preserve inline objects when decoding and re-encoding OPM
documents, but callers cannot construct that schema form directly without an
additional public representation. A future contract should decide whether
inline topic construction is intentionally unsupported or define a typed
reference that can hold either an ID or a topic without weakening the existing
`[]string` examples.

The dependency schema also has equivalent shorthand and long forms. A decoded
document can preserve its original form, and dependencies with locations
naturally require long form, but the documented `Dependency` fields do not let
a caller explicitly request long form without locations. A future contract
could either declare shorthand canonical in that case or expose an encoding
preference.

# Transcript entry contract

The transcript-entry schema requires `signed-by` to be a non-null email while
allowing `signature` to be null. This conflicts with the README's unsigned
transcript storage and with the signature-staleness behavior used by the other
signed model types, where both fields become null. The schema should allow
`signed-by` to be null and should require `signature` and `signed-by` to be
either both strings or both null.

The transcript entry documentation demonstrates `Sign` and `Verify` but does
not define its JSON and YAML methods, signature-access methods, or public error
behavior. The implementation necessarily provides entry and collection
encoding methods to satisfy the README, but it intentionally does not add the
`Signature`, `SignedBy`, and `SignatureKeyID` helpers documented for topic and
score-interpretation lists. The contract should state whether transcript
entries intentionally have the smaller signing API and should document
`ErrSignatureStale` and issuer-mismatch behavior if callers are expected to
inspect them.

The model specification does not define the canonical protected bytes signed
by different language implementations. It should specify field order,
timestamp normalization, JSON escaping, and the exact protected field set so
signatures interoperate. Only `topic-list-sources` is explicitly documented as
mutable. The specification should clarify whether `verification-url` is also
intended to be mutable; until then, the implementation uses the secure default
of including it in protected signature content.

The optional `$schema` property has no corresponding documented `Entry` field.
The current implementation preserves it privately during a decode and
re-encode round trip without expanding the public struct. The contract should
decide whether callers need a public field to construct entries containing
that property.

The checked-in transcript examples use abbreviated one-line placeholder
signatures that do not satisfy the transcript-entry schema's armored-signature
pattern. Valid armored fixtures, or a clear statement that these examples are
illustrative rather than schema-valid, would make them usable for conformance
testing.

# Score interpretation contract

The score-interpretation-list schema still restricts dependencies to URI
strings, while the package documentation and topic-list specification define
dependencies as `topic.Dependency` shorthand (`owner/name@version`) or
long-form objects. The implementation follows the package documentation and
bypasses only the dependency property during list-schema validation, then
performs the hostname, name, version, alias, and location validation
semantically. The model schema should be updated to reference the same
dependency definition as topic lists.

Operator IDs are shown in two forms in the package documentation: some
examples use a readable suffix such as `advanced-arithmetic`, while result
lookups use a complete schema operator key such as `any-advanced-subject` or
`at-least-2-pedagogy-skills`. The implementation accepts either form,
prefixing readable suffixes during encoding and using the resolved
schema-valid key in result maps. The public contract should explicitly name
that resolved-key behavior so callers do not have to infer it from examples.
