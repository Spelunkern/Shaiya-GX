# Weapon enchantments

Weapon glows read `Effect/Weapon.Seff` directly from the user's original archive.
No converted Phoenix assets or game textures are bundled. Phoenix's weapon
enchantment documentation and converter provide auxiliary references; native
EP5 routines `0x530790`, `0x530c70` and `0x4cba90` determine selection, parsing
and weapon normalization.

There are six visual tiers for each of fifteen weapon families. Levels below
+5 have no glow; +11 through +49 reuse +10; values at or above 50 represent
armor encoding and are excluded. Selection ignores the individual item model:

`record = 6 * (family - 1) + min(enchant, 10) - 4`

| Family | Weapon |
|---|---|
| 1 | One-handed sword |
| 2 | Two-handed sword |
| 3 | One-handed axe |
| 4 | Two-handed axe |
| 5 | Dual swords |
| 6 | Spear |
| 7 | One-handed mace |
| 8 | Two-handed hammer |
| 9 | Reverse dagger |
| 10 | Dagger |
| 11 | Javelin |
| 12 | Staff |
| 13 | Bow |
| 14 | Crossbow |
| 15 | Claws |

Alternate item types 45–65 normalize to these families. Shields do not emit.
Dual swords and claws attach independent emitters to both weapon bindings.
Each particle follows the current weapon pose, including its item-table binding;
it does not remain behind at a captured spell position.

The parser accepts SEFF versions 5–7, bounded UTF-16 filenames and complete
emitter fields. Unsupported emitters are reported rather than silently
substituted. The original local catalogue has 162 records; its ninety weapon
records contain 186 layers using eighteen textures. Remaining records are not
selected as weapon enchantments.

The renderer implements the point, radial, elongated and rotating emitter shapes
used by those records, their authored colors, sizes, speed, lifetime and alpha
ramp/fade. Native routines `0x52ebd0`, `0x52fa30`, `0x52fe60` and `0x5300a0`
provide the particle reference. Shared recipes and textures are cached; each
layer uses one instanced additive draw. A 2,000-particle cap per layer follows
the native pool. Effects respect the existing skill-effects visibility setting
and are omitted from shadow passes.

The local equipped level comes from the server-confirmed item's craft string.
Other players use the enchant byte already carried by native appearance
packets. Equipment moves and enchant acknowledgements rebuild the affected
appearance, including changes that preserve the same item type and model.
Success, failure, costs and consumption remain server-authoritative. Character
selection packets lack enchant metadata, so an unknown level is not invented.

GX Server's initial `0x0303` and later `0x0507` equipment updates now both
supply the raw craft level. Remote players therefore show their equipped glow
from their first appearance, including after entering visibility range. This
server correction is verified before any equipment move in the two-client test.

Validation covers all ninety recipes against original assets, both dual
bindings, level boundaries, visibility, actual rendered pixels and an eighty
weapon draw benchmark. The isolated two-client GX Server test exercises enchanting
an equipped +5 weapon, server acknowledgement and remote appearance updates,
then unequipping and re-equipping it. Captures and original data stay local.

Random births use stable analytical streams rather than the original random
generator. Exact particle-by-particle visual equivalence and all camera/animation
states still need comparison against the original running client. Phoenix's
additional dynamic lights are not part of this implementation.
