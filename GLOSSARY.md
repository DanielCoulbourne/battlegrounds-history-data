# Glossary

Every game word this format uses, in plain language. If you have never played
Hearthstone Battlegrounds, read this first.
[Section 2 of the specification](SPECIFICATION.md#2-the-game-for-programmers-who-have-not-played-it)
tells the same story in order rather than alphabetically.

**Activate.** An ability printed on some minions that you pay gold to use during
the recruit phase. It is a decision the player makes, so the format records it as
an action.

**Anomaly.** A rule change that applies to a whole lobby for a whole game, in the
seasons that have them. Recorded on the `game` object.

**Armor.** Extra starting health, granted by some heroes. Most effects treat it
as health. The format keeps it separate wherever the game does.

**Attack.** The left-hand number on a minion. It is how much damage the minion
deals when it fights.

**Battlecry.** An effect that fires when you play a card from your hand.

**Board.** Your row of up to seven minions. Their left-to-right order matters:
minions attack in that order, and some effects care about who is next to whom.

**Buddy.** A hero-specific minion from an older season. The format has an
`offerType` for it so recordings of those seasons stay expressible.

**Cleave.** An attack that also hits the minions either side of its target.

**Combat phase.** The fight. The game pairs you with another player, and your two
boards fight without either of you touching them. See also *recruit phase*.

**Dark Gift.** A season-specific enchantment attached to a minion. In this format
it is an `enchantment` on a card, and choosing one is an `offer` with
`offerType: darkGift`.

**Deathrattle.** An effect that fires when a minion dies.

**Discover.** The game shows you a few cards and you keep one. In this format it
is an `offer` event followed by a `choose` action.

**Divine Shield.** The first hit this minion takes is swallowed whole: it loses
no health, and the shield is gone. It swallows poison along with the hit.

**Duos.** A game mode with four teams of two. Teammates can pass cards to each
other.

**Enchantment.** Something attached to a card that changes it — a buff from a
spell, a Dark Gift, a lasting effect. The format lists them on the card, with the
card's numbers already including them.

**Freeze.** Hold the shop over to your next turn instead of getting a new one.

**Ghost.** A copy of a knocked-out player's board. When an odd number of players
is left, one of them fights a ghost. The player who once owned the board is
already out, so nothing that happens to the copy affects them.

**Gold.** What you spend in the recruit phase. You are given a fresh amount at
the start of every turn, and it does not carry over.

**Golden.** A stronger version of a card, made by collecting three copies of it.
See *triple*.

**Hand.** Cards you own but have not put down. It holds up to ten.

**Health.** For a minion, the right-hand number: the damage it can take before it
dies. For a player, the total that reaches 0 when they are knocked out.

**Hero.** The character you play. It sets your starting health and usually comes
with a hero power.

**Hero power.** A special ability printed on your hero. Some fire on their own;
some you press.

**Immune.** Takes no damage.

**Lobby.** One game: eight players, playing until one is left.

**Magnetize.** Fuse a Mech card from your hand onto a Mech already on your board.
The two become one minion carrying the stats and abilities of both.

**Mega-Windfury.** This minion attacks four times per turn. See *Windfury*.

**Minion.** A creature that fights for you.

**Minion type.** A family a minion belongs to: Beast, Demon, Dragon, Elemental,
Mech, Murloc, Naga, Pirate, Quilboar, Undead. Players call it a **tribe**. Each
lobby uses only some of them, so the card pool changes from game to game. A card
of type "all" counts as every type at once.

**Placement.** Where a player finished. 1 is the winner, 8 is out first.

**Poisonous.** Any damage this minion deals destroys what it hits, however much
health that had. See *Venomous*.

**Recruit phase.** The shopping half of a turn. Only you can see it. You buy,
play, sell and rearrange cards, and decide when to end it. See also *combat
phase*.

**Reborn.** When this minion dies, it comes back once — as the printed card, at 1
health, with none of the buffs it had.

**Roll.** Pay to replace every card in the shop. Also called refreshing.

**Seat.** A player's position at the table. The format keeps `seat` separate from
a player's `id`, because seat numbers are not stable across producers.

**Shop.** The row of cards you can buy, also called the tavern. It grows as your
tavern tier rises.

**Standings.** The leaderboard: every living player's hero, health and tavern
tier. It is all a player normally sees of the other seats.

**Start of Combat.** An effect that fires before the first attack of a fight.

**Stealth.** This minion cannot be chosen as a target until it attacks.

**Taunt.** This minion must be attacked before anything else on its board.

**Tavern.** Another word for the shop.

**Tavern tier.** A number from 1 to 6 saying how strong the cards in your shop
can be. You pay gold to raise it, and the price falls by 1 for every turn you do
not.

**Trinket.** A keepsake you pick up mid-game that changes your whole game. A
lesser trinket is offered earlier and a greater one later.

**Triple.** Collect three copies of the same card and they merge into one golden
copy, which is stronger. Making one also lets you Discover a card from a tier
above your own.

**Venomous.** Like Poisonous, but spent after one kill.

**Windfury.** This minion attacks twice per turn.
