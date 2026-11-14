# Troubleshooting and worked examples

## Linking example

For a lapis with ReqIg 31, without a blessing modifier, the base chance is
`289 - 8*31 = 41%`. The common linking hammer adds 20 percentage points, giving
61%. Its gold cost is 4,095. A valid attempt consumes one lapis, one selected
hammer and the gold, whether it succeeds or fails. Existing equipment and gems
survive failure. Without a selected hammer the same example has 41% chance.

If the lapis requires two capacity units and equipment has only one free unit,
the attempt is refused before consumption. Six visible/stored gem entries do
not imply six free capacity units: each lapis's Slot value matters.

## Extraction example

Select one installed lapis, then place a small extraction hammer (30%). The
quote shows the selected lapis's native gold cost. Success returns that lapis to
inventory; failure destroys that lapis. Both outcomes leave the equipment,
recreation, enchantment and other gems intact. An extraction hammer does not add
30 points to the linking formula: it defines the extraction chance directly.

## Enhancement example

An unenchanted weapon attempts +1 with 90% chance. An unenchanted armor item
attempts +1 with 99%. The selected lapisia sets the native gold-table entry,
while the current level and weapon/armor category select the compiled chance.
Each attempt consumes exactly one required material and one lapisia,
without a quantity selector or confirmation. The chance appears above Enchant. Failure never destroys equipment or removes the existing level.

## Recreation example

For an eligible weapon with Server 3 and ReqWis 10, recreation chooses three
different attributes from STR/DEX/REC/INT/WIS/LUC and rolls each from 1 to 10.
The remaining attributes become zero. It consumes one rune, preserves gems and
enchantment and does not ask for confirmation. The new result can be worse.

## Why is a gem rejected when its category appears in the tooltip?

The category only establishes native family compatibility. Check equipment
capacity, duplicate TypeID, item count, the service view, required materials and
whether the target has the required server definition fields. The world server
revalidates all of this, including when the client data is stale.

## Why does a placed item remain after an attempt?

That is intentional. Placed items reference inventory slots and refresh from
confirmed state. Remaining consumable stacks support continuous operations.
Click a placed slot to deselect it without consuming it. Only the exhausted
selection disappears; it does not eject the equipment.

## Why does changing ItemEnchant.ini do nothing?

It is retired. Edit `server/internal/world/enchant_rules.go`, validate the cap,
rates and bonus tables, rebuild `gx_world`, then restart the affected world
service. The historical INI under `server/docs/reference/` is research material.
See [Blacksmith](blacksmith.md) for the complete tables and encoding limits.

## Why does the client disagree with the server about an item?

Check that SQL Items and the external Item.SData were updated together, and that
both components restarted. The client stack-limit export must match SQL Count.
Mall content additionally requires matching Cash.SData and SQL ProductList.
The server always decides actual grants, costs, stats and consumption.

## Why are products free, or a stack capped at 255?

The current testing catalog intentionally uses zero prices. That does not make
blacksmith operations free. Stacks use native one-byte count packets; wider
internal arithmetic does not raise the supported quantity in an inventory slot.
Matching client/server revision is required. See [Item Mall](item-mall.md).

## Why do drops have bonuses before visiting the blacksmith?

New eligible mob/sequence equipment gets a valid recreation roll at ground-item
creation. Pickup saves that same Craftname. Dropping and picking up owned gear
preserves it and cannot be used to reroll for free.

## Which documentation applies to an older release?

The wiki describes current development behavior. Released binaries can lag
behind it. Compare the release notes and use matching client and world versions.
A wiki update does not install code, replace a runtime database or update assets.
Historical reverse-engineering notes are evidence, not configuration.
