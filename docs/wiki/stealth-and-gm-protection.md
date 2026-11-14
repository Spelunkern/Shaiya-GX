# Stealth and GM protection

GX controls concealment and immunity on the world server. Editing a client flag,
keeping an old target ID or sending an attack packet does not bypass these rules.

## Ordinary stealth

An active buff with `TypeEffect = 4`, `TypeDetail = 88` grants stealth. Skill
learning, costs, cooldown and duration still come from the normal Skills data.
Stealth does not change movement speed, force walking or prevent running.

The caster and friendly faction members see a translucent character. Enemy
faction clients receive a native character-leave packet and stop receiving its
entry, movement, appearance, cast and combat/effect broadcasts. A permitted viewer
that newly enters range receives the current concealment state along with the
character. Leaving stealth reintroduces the character even when nobody moves.

An accepted normal attack, offensive skill or debuff ends stealth, including an
attack that misses. Skills with a cast time reveal at cast start, after target,
range and resource validation. Rejected attacks do not reveal the player. Actual
HP, MP or SP damage also ends stealth, including periodic damage, reflection and
area hits. Friendly buffs and healing do not themselves break ordinary stealth.
A native character leave also clears that client's selected target.

Monsters exclude concealed characters from acquisition, current-target checks,
forced targeting, strongest-aggro targeting and weakest-participant targeting.
A monster losing its target selects another eligible combat participant from its
aggro ledger. If none remains, it returns home instead of attacking a nearby
bystander. Pending primary monster attacks are discarded if the target has become
concealed; area damage can still hit a concealed participant and reveal it.
Buff detail 89 separately prevents monster targeting without concealing the
character from players.

NPC service actors do not implement a separate attack AI. Hostile creatures use
the monster combat path above; patrol and service interaction remain independent.

## GM defaults and commands

Each character entry on a GM account starts invisible and invulnerable. These are
independent session states, not permanent character or account edits.

| Command | Effect |
|---|---|
| `/char off` | Disables GM invisibility. An ordinary active stealth buff can still conceal the character. |
| `/char on` | Restores GM invisibility and stops an ongoing automatic attack/cast. |
| `/attack off` | Disables GM invulnerability. It does not change invisibility. |
| `/attack on` | Restores GM invulnerability and removes adverse skill effects; positive buffs remain. It does not change invisibility. |

A hidden GM is invisible to ordinary players of **both factions**. Other GMs and
the GM themself see the translucent character. Party and raid membership is
retained, but hidden coordinates are redacted from roster/join records delivered
to unauthorized viewers. Hidden GMs cannot be newly selected for trade by an
unauthorized player.

Invulnerability rejects normal attacks, offensive skills, reflected hits,
periodic damage, sacrifice damage and external skill effects, including allied
healing, cures and buffs. The GM's own positive buffs, healing and cures remain
usable, with normal costs, cooldowns, server confirmations, effects and sound.
The session checks caster identity again when processing queued effects; an
external caster cannot bypass immunity by marking an effect friendly.

Restoring invulnerability removes adverse effects and retains positive buffs.
Positive saved buffs are restored on login; saved adverse effects are discarded
when entering as a protected GM. Saved buffs do not retain caster provenance,
so existing positive buffs are preserved regardless of their original caster.
Direct and area attack selection still excludes immune targets.

GM invisibility is a persistent command mode: attacking, casting and receiving
damage do not disable `/char on`. Only `/char off` disables that mode. Ordinary
skill-based stealth still breaks on attacks and actual damage. Both GM modes
remain independent. Self and authorized GM observers retain normal confirmed
combat audio and effects; hidden entities and their actor packets are withheld
from unauthorized viewers.

## Original visual and GX transport

The original server applies shape 13 for buff detail 88. Original client
`Game.exe` handler `0x00591FE0` dispatches native `0x051D` to
`0x00561D70` / `0x004CC130`. Its shape-13 branch sets the permitted-viewer hidden
render flag at character offset `0x288`. The body renderer `0x00414FB0`, branch
near `0x004151C5`, uses texture factor `0x64FFFFFF`: white with alpha **100/255**.
This is an intermediate texture factor, not a complete specification of the
final composited appearance. The initial GX translation multiplied texture alpha by concealment opacity.
Several otherwise opaque clothing textures contain zero alpha, making the body
nearly disappear. The original pass selects texture-factor alpha instead.
GX likewise replaces unused material alpha after any authored cutout test. Following visual review, GX uses
65% opacity, preserving original RGB colors without grayscale or tint. Body,
equipment, cape and mount share that presentation; weapon glow and shadows are
suppressed. Source-texture cutouts are tested before ghost opacity. The ghost
pass retains clockwise front faces and depth writes, so the rear of a head does
not expose the face through it. Culling/blending state is restored afterward.

GX carries authorized concealment state in extension `0x7F02`: character ID
`u32` and concealed `u8` (0 or 1), following the normal frame/opcode header. Only
permitted viewers receive this metadata. Forbidden viewers receive native
`0x0202` leave; the server withholds the entity itself. Native mount packet
layouts and the existing GM `0x7F01` format are unchanged. Use matching GX client
and world builds. No database migration or client asset change is required.

The original forced-walk behavior is intentionally omitted, as requested for GX.
Enemy stealth detection/reveal skills are not introduced by this change; GMs can
observe concealed characters through their server role.
