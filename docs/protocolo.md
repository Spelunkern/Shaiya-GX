# Protocolo EP5.4

Generado por `tools/protomap.py`; no editar a mano.

- **Recibidos:** los 406 opcodes que despacha el cliente original (`fun_0059cf70`, `network/dispatcher.cpp` del decomp).
- **Decodificado:** sí = `internal/game` lo aplica; no = se cuenta y se ignora.
- **Enviados:** los writers del original verificados en Ghidra (rango `0x597f60..0x59a000`): 0x5980e0, 0x598180, 0x5981b0, 0x5981e0, 0x598210, 0x598460, 0x598980, 0x598ab0, 0x598b00, 0x5992a0, 0x5992f0, 0x599440, 0x599480, 0x599a30, 0x599a60.

Recibidos: 406. Decodificados: 27.

| Opcode | Handler del cliente | Nombre | Decodificado |
|---|---|---|---|
| 0x0002 | `virtual_no_args` |  | no |
| 0x0004 | `fun_0058d830` |  | no |
| 0x0101 | `fun_0058d860` | CharacterList | sí |
| 0x0102 | `fun_0058db20` | CreateCharacter | sí |
| 0x0103 | `fun_0058db50` | DeleteCharacter | sí |
| 0x0104 | `fun_0058db90` | SelectCharacter | sí |
| 0x0105 | `fun_0058dbd0` | CharacterDetails | sí |
| 0x0106 | `fun_0058dd30` | CharacterItems | no |
| 0x0107 | `fun_004015a0` | Logout | sí |
| 0x0108 | `fun_0058e020` | CharacterSkills | no |
| 0x0109 | `fun_0058e0e0` | AccountFaction | sí |
| 0x010A | `fun_00591d70` | CharacterActiveBuffs | no |
| 0x010B | `fun_00591dd0` | CharacterSkillBar | no |
| 0x010D | `virtual_no_args` |  | no |
| 0x010E | `dispatch_packet_u8_to_session_slot_050` | RenameCharacter | no |
| 0x010F | `fun_00591e30` | RestoreCharacter | sí |
| 0x0110 | `dispatch_packet_u8_to_session_slot_040` |  | no |
| 0x0111 | `fun_0058e190` |  | no |
| 0x0112 | `fun_0058e120` |  | no |
| 0x0113 | `virtual_no_args` |  | no |
| 0x0116 | `fun_0058e240` |  | no |
| 0x0117 | `fun_00594cd0` |  | no |
| 0x0201 | `fun_0058e280` | CharacterEnteredMap | sí |
| 0x0202 | `dispatch_packet_u32_to_session_slot_064` | UserLeave | sí |
| 0x0204 | `fun_0058e510` | ItemBagToBag | no |
| 0x0205 | `fun_0058eb30` | AddItem | no |
| 0x0206 | `fun_0058eca0` | RemoveItem | no |
| 0x0207 | `fun_0058ed20` | ExpGain | no |
| 0x0208 | `fun_0058eda0` | SetStatPoint | no |
| 0x0209 | `fun_0058ee30` | LearnNewSkill | no |
| 0x020A | `fun_00597a30` | CharacterEnteredPortal | no |
| 0x020B | `fun_0058ef60` | CharacterMapTeleport | sí |
| 0x020C | `fun_00595630` | Gatekeeper | no |
| 0x020D | `fun_0058efe0` |  | no |
| 0x020E | `fun_0058f030` | UserKillCountUpdate | no |
| 0x020F | `fun_0058f080` |  | no |
| 0x0210 | `dispatch_packet_u8_to_session_slot_08c` |  | no |
| 0x0211 | `fun_0058eed0` | BlessInit | no |
| 0x0212 | `virtual_no_args` | AutoAttackStop | no |
| 0x0213 | `dispatch_packet_u32_to_session_slot_098` | SetMoney | no |
| 0x0214 | `fun_00595680` | StatsReset | no |
| 0x0215 | `fun_00595740` |  | no |
| 0x0216 | `fun_00595bf0` | UseVehicle | no |
| 0x0217 | `dispatch_packet_u32_to_session_slot_4dc` |  | no |
| 0x0218 | `fun_00595d20` |  | no |
| 0x0219 | `fun_00595d70` |  | no |
| 0x021A | `fun_00595dc0` |  | no |
| 0x021B | `fun_00596100` |  | no |
| 0x021C | `fun_00595c70` |  | no |
| 0x021D | `dispatch_packet_u32_to_session_slot_4e4` |  | no |
| 0x021E | `dispatch_packet_u8_to_session_slot_4e8` |  | no |
| 0x0220 | `fun_00596150` |  | no |
| 0x0221 | `dispatch_packet_u32_to_session_slot_524` |  | no |
| 0x0222 | `fun_00596210` |  | no |
| 0x0223 | `dispatch_packet_u32_to_session_slot_530` |  | no |
| 0x0224 | `fun_00596290` |  | no |
| 0x0225 | `fun_00596950` | TeleportSavePositionList | no |
| 0x0226 | `fun_0058fc50` | ChangeAppearance | no |
| 0x0228 | `virtual_no_args` |  | no |
| 0x0230 | `dispatch_packet_u16_to_session_slot_058` |  | no |
| 0x0301 | `fun_0058f0d0` |  | no |
| 0x0302 | `fun_0058f200` |  | no |
| 0x0303 | `fun_0058e360` | CharacterShape | no |
| 0x0304 | `fun_0058f260` | TargetMobGetState | no |
| 0x0305 | `fun_0058f2d0` | TargetMobHpUpdate | no |
| 0x0307 | `fun_0058f150` |  | no |
| 0x0401 | `fun_0058f320` | MapAddItem | no |
| 0x0402 | `dispatch_packet_u32_to_session_slot_0d0` | MapRemoveItem | no |
| 0x0403 | `fun_0058f410` |  | no |
| 0x0404 | `dispatch_packet_u32_to_session_slot_0d8` | WorldDay | no |
| 0x0405 | `fun_00591e70` |  | no |
| 0x0406 | `dispatch_packet_u32_to_session_slot_218` | CharacterReviveInit | no |
| 0x0451 | `fun_0058f490` |  | no |
| 0x0501 | `fun_0058f4f0` | CharacterMove | sí |
| 0x0502 | `fun_0058f640` | CharacterUserAutoAttack | no |
| 0x0503 | `fun_0058f6d0` | CharacterMobAutoAttack | no |
| 0x0504 | `fun_0058f760` | CharacterDeath | no |
| 0x0505 | `fun_0058f7c0` | CharacterReviveHitpoints | no |
| 0x0506 | `fun_0058f830` | CharacterMotion | no |
| 0x0507 | `fun_0058f980` | CharacterEquipment | no |
| 0x0508 | `fun_0058fa30` | LevelUpSelf | no |
| 0x0509 | `fun_0058fbc0` |  | no |
| 0x050B | `fun_0058fcd0` | CharacterMaxEnergy | no |
| 0x050C | `fun_0058fd30` | SkillUsedSpMp | no |
| 0x050D | `fun_0058fd80` | BuffAdd | no |
| 0x050E | `dispatch_packet_u32_to_session_slot_13c` | BuffRemove | no |
| 0x050F | `fun_0058fe10` | SkillKeep | no |
| 0x0510 | `fun_0058fea0` | SkillCastStarted | no |
| 0x0511 | `fun_0058ff10` | SkillUseNoTarget | no |
| 0x0512 | `dispatch_packet_11_mixed_fields_to_session_slot_150` |  | no |
| 0x0513 | `fun_00590120` | SkillPartyMember | no |
| 0x0514 | `fun_005901e0` |  | no |
| 0x0515 | `fun_00590280` |  | no |
| 0x0516 | `fun_00590300` |  | no |
| 0x0517 | `fun_00590370` | SkillUse | no |
| 0x0518 | `dispatch_packet_11_mixed_fields_to_session_slot_168` |  | no |
| 0x0519 | `fun_00590520` |  | no |
| 0x051A | `fun_005905e0` | SkillExtraDamage | no |
| 0x051B | `fun_00590680` |  | no |
| 0x051C | `fun_00591a30` | CharacterDeathState | no |
| 0x051D | `fun_00591fe0` | CharacterMountShape | no |
| 0x051E | `fun_0058fac0` | LevelUpOther | no |
| 0x051F | `fun_0058fb50` | CharacterMaxHpMpSp | no |
| 0x0520 | `fun_00590700` | PartyBadge | no |
| 0x0521 | `fun_00595090` | CharacterCurrentHitpoints | no |
| 0x0522 | `fun_005950f0` |  | no |
| 0x0523 | `fun_005957c0` |  | no |
| 0x0524 | `fun_00592030` |  | no |
| 0x0525 | `dispatch_packet_u16_to_session_slot_538` |  | no |
| 0x0526 | `dispatch_packet_12_dwords_to_session_slot_0fc` | CharacterAdditionalStats | no |
| 0x0527 | `virtual_no_args` |  | no |
| 0x0551 | `fun_00592090` | CharacterResurrect | no |
| 0x0554 | `fun_0058f5b0` | ItemRepair | no |
| 0x0555 | `dispatch_packet_u16_to_session_slot_128` |  | no |
| 0x0558 | `fun_0058ffd0` | SkillDetectInfo | no |
| 0x0601 | `fun_00590b50` | MobEnter | sí |
| 0x0602 | `dispatch_packet_u32_to_session_slot_194` | MobLeave | sí |
| 0x0603 | `fun_00590c00` | MobMove | sí |
| 0x0604 | `dispatch_packet_u32_to_session_slot_19c` |  | no |
| 0x0605 | `fun_00590ca0` | MobAttackCharacter | no |
| 0x0606 | `fun_00590d30` | MobDeath | no |
| 0x0607 | `fun_00590df0` | MobPeriodicDamage | no |
| 0x0608 | `fun_00590d90` |  | no |
| 0x0609 | `fun_00591a90` |  | no |
| 0x060B | `dispatch_packet_9_mixed_fields_to_session_slot_17c` |  | no |
| 0x060C | `dispatch_packet_12_mixed_fields_to_session_slot_180` |  | no |
| 0x060D | `fun_00590970` |  | no |
| 0x060E | `fun_00590a30` |  | no |
| 0x060F | `fun_00590ad0` |  | no |
| 0x0610 | `fun_00590750` | MobCurrentHitpoints | no |
| 0x0701 | `fun_00590e80` |  | no |
| 0x0702 | `fun_00590ef0` | ShopBuy | no |
| 0x0703 | `fun_00590f90` | ShopSell | no |
| 0x0711 | `fun_00591030` | WarehouseItemList | no |
| 0x0801 | `dispatch_packet_13_mixed_fields_to_session_slot_1c8` | GemAdd | no |
| 0x0802 | `fun_005918a0` | GemRemove | no |
| 0x0803 | `fun_00596e80` |  | no |
| 0x0804 | `fun_00596f00` |  | no |
| 0x0805 | `fun_005917b0` | EnchantAdd | no |
| 0x0806 | `fun_00591170` | ItemCompose | no |
| 0x0807 | `fun_005911e0` |  | no |
| 0x0808 | `fun_005915f0` |  | no |
| 0x080B | `fun_00591330` |  | no |
| 0x080C | `fun_00591330` |  | no |
| 0x080D | `fun_00591490` |  | no |
| 0x0901 | `fun_00591b50` | QuestList | no |
| 0x0902 | `fun_00591bb0` | QuestStart | no |
| 0x0903 | `dispatch_packet_10_mixed_fields_to_session_slot_200` | QuestEnd | no |
| 0x0905 | `fun_00591ce0` | QuestCounter | no |
| 0x0906 | `fun_00591af0` | QuestFinishedList | no |
| 0x0907 | `dispatch_packet_u16_to_session_slot_208` | QuestChoose | no |
| 0x0A01 | `dispatch_packet_u32_to_session_slot_29c` | ExchangeRequest | no |
| 0x0A03 | `fun_00592cc0` | ExchangeStart | no |
| 0x0A04 | `dispatch_packet_u8_to_session_slot_2a4` | ExchangeEnd | no |
| 0x0A05 | `fun_00592d40` | ExchangeReady | no |
| 0x0A06 | `fun_00596fd0` | ExchangeAdd | no |
| 0x0A07 | `fun_00592de0` | ExchangeRemove | no |
| 0x0A08 | `fun_00592e30` | ExchangeAddMoney | no |
| 0x0A09 | `fun_00597040` | ExchangeAddDest | no |
| 0x0A0A | `fun_00592d90` | ExchangeConfirm | no |
| 0x0B01 | `fun_00592480` | PartyRoster | no |
| 0x0B02 | `fun_00592620` | PartyInvite | no |
| 0x0B03 | `fun_00592650` | PartyAnswer | no |
| 0x0B04 | `fun_005926a0` | PartyJoined | no |
| 0x0B05 | `dispatch_packet_u32_to_session_slot_260` | PartyLeft | no |
| 0x0B06 | `dispatch_packet_u32_to_session_slot_264` | PartyExpelled | no |
| 0x0B07 | `dispatch_packet_u32_to_session_slot_268` | PartyLeader | no |
| 0x0B08 | `fun_00592bd0` |  | no |
| 0x0B09 | `fun_0058d5c0` | PartyInviteName | no |
| 0x0B0B | `fun_00596740` | RaidRoster | no |
| 0x0B0C | `fun_005965f0` | RaidJoined | no |
| 0x0B0D | `dispatch_packet_u32_to_session_slot_544` | PartyLeaveAlt | no |
| 0x0B0E | `fun_00596380` | RaidConvert | no |
| 0x0B0F | `dispatch_packet_u32_to_session_slot_54c` | RaidDivision | no |
| 0x0B10 | `dispatch_packet_u8_to_session_slot_550` | RaidAutoJoin | no |
| 0x0B12 | `fun_00596490` | RaidSwap | no |
| 0x0B13 | `dispatch_packet_u32_to_session_slot_55c` | RaidSubLeader | no |
| 0x0B14 | `dispatch_packet_u32_to_session_slot_568` | RaidInvite | no |
| 0x0B15 | `fun_00592650` | RaidAnswer | no |
| 0x0B16 | `fun_0058d6f0` | RaidDisband | no |
| 0x0B17 | `dispatch_packet_u32_to_session_slot_558` | RaidLeader | no |
| 0x0B18 | `fun_0058d710` | RaidInviteName | no |
| 0x0B19 | `dispatch_packet_u32_to_session_slot_564` | RaidError | no |
| 0x0B1A | `dispatch_packet_u32_to_session_slot_560` | RaidExpel | no |
| 0x0B1B | `fun_0059cf10` |  | no |
| 0x0B1C | `fun_00592c30` |  | no |
| 0x0C01 | `fun_00592870` |  | no |
| 0x0C02 | `fun_005928d0` |  | no |
| 0x0C03 | `fun_00592930` | PartyPools | no |
| 0x0C04 | `fun_005929a0` |  | no |
| 0x0C05 | `fun_00592a00` |  | no |
| 0x0C06 | `fun_00592a60` |  | no |
| 0x0C07 | `fun_00592a90` |  | no |
| 0x0C08 | `fun_00592b10` |  | no |
| 0x0C09 | `fun_00592b80` | PartyLevel | no |
| 0x0C0A | `fun_00592870` |  | no |
| 0x0C0B | `fun_005928d0` |  | no |
| 0x0C0C | `fun_00592930` | RaidPools | no |
| 0x0C0D | `fun_00592b10` |  | no |
| 0x0C0E | `fun_005929a0` |  | no |
| 0x0C10 | `fun_00592a00` |  | no |
| 0x0C11 | `fun_00592a60` |  | no |
| 0x0C12 | `fun_00592a90` |  | no |
| 0x0C13 | `fun_00592b80` |  | no |
| 0x0C14 | `fun_00592bd0` |  | no |
| 0x0D01 | `fun_00594620` | GuildCreateFailed | no |
| 0x0D02 | `fun_005945c0` |  | no |
| 0x0D03 | `dispatch_packet_u8_to_session_slot_3bc` | GuildDisband | no |
| 0x0D04 | `dispatch_packet_u32_to_session_slot_3b8` | GuildDisbandAgree | no |
| 0x0D05 | `dispatch_packet_u32_to_session_slot_3c0` | GuildInvite | no |
| 0x0D06 | `fun_00594740` | GuildInviteAnswer | no |
| 0x0D07 | `dispatch_packet_u8_to_session_slot_3c8` | GuildApply | no |
| 0x0D08 | `fun_005947c0` | GuildJoinResult | no |
| 0x0D09 | `dispatch_packet_u8_to_session_slot_3d0` | GuildLeave | no |
| 0x0D0A | `fun_00594860` | GuildExpel | no |
| 0x0D0B | `fun_005948b0` |  | no |
| 0x0D0C | `fun_00594940` | GuildStatus | no |
| 0x0D0D | `virtual_no_args` | GuildListBegin | no |
| 0x0D0E | `virtual_no_args` | GuildListEnd | no |
| 0x0D0F | `fun_00594990` |  | no |
| 0x0D10 | `fun_005949f0` |  | no |
| 0x0D11 | `dispatch_packet_u32_to_session_slot_3f0` | GuildDisbanded | no |
| 0x0D12 | `fun_00594ac0` | GuildOnlineList | no |
| 0x0D13 | `fun_00594b20` | GuildOfflineList | no |
| 0x0D14 | `fun_00594b80` | GuildMemberAdded | no |
| 0x0D15 | `dispatch_packet_u32_to_session_slot_404` |  | no |
| 0x0D16 | `fun_00594c00` | GuildApplicantList | no |
| 0x0D17 | `fun_00594c60` | GuildApplicant | no |
| 0x0D18 | `dispatch_packet_u32_to_session_slot_410` | GuildApplicantRemoved | no |
| 0x0D19 | `fun_00594a60` |  | no |
| 0x0D20 | `fun_00597870` | GuildRemark | no |
| 0x0D21 | `fun_005975c0` | GuildCreate | no |
| 0x0D22 | `fun_005945c0` | GuildCreateAgree | no |
| 0x0D23 | `dispatch_packet_u32_to_session_slot_5ec` | GuildEtin | no |
| 0x0D24 | `fun_005977d0` |  | no |
| 0x0D25 | `fun_00597940` |  | no |
| 0x0D26 | `fun_005976d0` | GuildRankingNotice | no |
| 0x0D27 | `fun_00597640` | GuildRankingPoints | no |
| 0x0D28 | `fun_005979b0` | GuildBankResult | no |
| 0x0D29 | `fun_00597250` | GuildBankList | no |
| 0x0D2F | `fun_00597720` | GuildList | no |
| 0x0D30 | `fun_00597830` | GuildNew | no |
| 0x0D31 | `fun_00597380` | GuildBankPlaced | no |
| 0x0D32 | `fun_005974a0` | GuildBankTaken | no |
| 0x0D34 | `fun_005978f0` |  | no |
| 0x0D35 | `fun_00597780` |  | no |
| 0x0D36 | `dispatch_packet_u8_to_session_slot_604` |  | no |
| 0x0D37 | `fun_00597a80` | GuildRankingList | no |
| 0x0D38 | `dispatch_packet_u32_to_session_slot_60c` |  | no |
| 0x0E01 | `fun_00592f50` | MapNpcEnter | sí |
| 0x0E02 | `dispatch_packet_u32_to_session_slot_2d4` | MapNpcLeave | sí |
| 0x0E03 | `fun_00593020` | MapNpcMove | no |
| 0x0E04 | `dispatch_packet_u32_to_session_slot_2dc` | NpcTalk | no |
| 0x0E05 | `fun_005930d0` |  | no |
| 0x0E06 | `fun_00593140` |  | no |
| 0x0F01 | `fun_00592e80` |  | no |
| 0x0F02 | `dispatch_packet_u8_to_session_slot_2cc` |  | no |
| 0x1101 | `fun_005931b0` | ChatNormal | sí |
| 0x1102 | `fun_00593230` | ChatWhisper | sí |
| 0x1103 | `fun_005932d0` | ChatWorld | sí |
| 0x1104 | `fun_00593360` | ChatGuild | sí |
| 0x1105 | `fun_005933f0` | ChatParty | sí |
| 0x1106 | `dispatch_packet_u8_to_session_slot_2f0` | ChatError | no |
| 0x1107 | `fun_005934a0` | ChatShout | sí |
| 0x1108 | `fun_00593510` | ChatWorldShout | sí |
| 0x1109 | `fun_005935a0` |  | no |
| 0x110A | `fun_00593640` |  | no |
| 0x110B | `fun_00593690` |  | no |
| 0x1111 | `fun_00593700` | ChatMap | sí |
| 0x1112 | `fun_00596310` |  | no |
| 0x2101 | `fun_00594d20` |  | no |
| 0x2102 | `fun_00594d80` |  | no |
| 0x2103 | `fun_00594dd0` |  | no |
| 0x2104 | `fun_00594e30` |  | no |
| 0x2201 | `fun_00595140` |  | no |
| 0x2202 | `fun_0058d6c0` |  | no |
| 0x2203 | `dispatch_packet_u8_to_session_slot_460` |  | no |
| 0x2204 | `fun_00595250` |  | no |
| 0x2205 | `dispatch_packet_u32_to_session_slot_468` |  | no |
| 0x2206 | `fun_005952e0` |  | no |
| 0x2207 | `fun_00595350` |  | no |
| 0x2208 | `fun_005953a0` |  | no |
| 0x2209 | `fun_00595440` |  | no |
| 0x220A | `dispatch_packet_u32_to_session_slot_47c` |  | no |
| 0x220B | `fun_005954c0` |  | no |
| 0x220C | `dispatch_packet_u8_to_session_slot_488` |  | no |
| 0x220D | `fun_00595590` |  | no |
| 0x220E | `dispatch_packet_u8_to_session_slot_484` |  | no |
| 0x2301 | `virtual_no_args` |  | no |
| 0x2302 | `virtual_no_args` |  | no |
| 0x2303 | `virtual_no_args` |  | no |
| 0x2304 | `virtual_no_args` |  | no |
| 0x2305 | `fun_00595840` |  | no |
| 0x2306 | `dispatch_packet_u8_to_session_slot_4b4` |  | no |
| 0x2307 | `fun_005958e0` |  | no |
| 0x2308 | `fun_00595930` |  | no |
| 0x2309 | `fun_00595990` |  | no |
| 0x230A | `virtual_no_args` |  | no |
| 0x230B | `fun_005959e0` |  | no |
| 0x230C | `virtual_no_args` |  | no |
| 0x230D | `fun_00595a40` |  | no |
| 0x230E | `fun_00595ba0` |  | no |
| 0x2401 | `fun_00595e10` |  | no |
| 0x2402 | `fun_00595e60` |  | no |
| 0x2403 | `fun_00595eb0` |  | no |
| 0x2404 | `virtual_no_args` |  | no |
| 0x2405 | `dispatch_packet_u8_to_session_slot_4fc` |  | no |
| 0x2406 | `fun_00595f30` |  | no |
| 0x2407 | `fun_00596f50` |  | no |
| 0x2408 | `dispatch_packet_u8_to_session_slot_2c4` |  | no |
| 0x2409 | `fun_00592d40` |  | no |
| 0x240A | `fun_00596fd0` |  | no |
| 0x240B | `fun_00592de0` |  | no |
| 0x240C | `fun_00592e30` |  | no |
| 0x240D | `fun_00597040` |  | no |
| 0x2501 | `fun_00595f80` |  | no |
| 0x2502 | `fun_00595fd0` |  | no |
| 0x2503 | `fun_00596020` |  | no |
| 0x2504 | `virtual_no_args` |  | no |
| 0x2505 | `dispatch_packet_u8_to_session_slot_514` |  | no |
| 0x2506 | `fun_005960b0` |  | no |
| 0x2601 | `fun_005970e0` | AccountPoints | no |
| 0x2602 | `fun_00597160` | PointBuy | no |
| 0x2603 | `fun_00596b90` | PointGiftSend | no |
| 0x2604 | `fun_00596c30` |  | no |
| 0x2605 | `dispatch_packet_u32_to_session_slot_5b4` | CashPoint | no |
| 0x2606 | `fun_00596aa0` | PointStoredItemList | no |
| 0x2607 | `fun_00596b30` | PointStoredItemGet | no |
| 0x2609 | `fun_00596d10` |  | no |
| 0x2701 | `fun_00597b10` |  | no |
| 0x2702 | `fun_00597c40` |  | no |
| 0x2703 | `dispatch_packet_u8_to_session_slot_618` |  | no |
| 0x2704 | `fun_00597ae0` |  | no |
| 0x3101 | `dispatch_packet_u8_to_session_slot_428` |  | no |
| 0x3102 | `fun_00594ec0` |  | no |
| 0x3103 | `fun_00594f20` |  | no |
| 0x3104 | `dispatch_packet_u8_to_session_slot_434` |  | no |
| 0x3105 | `virtual_no_args` |  | no |
| 0x3106 | `dispatch_packet_u32_to_session_slot_43c` |  | no |
| 0x3107 | `fun_00595000` |  | no |
| 0x3108 | `dispatch_packet_u8_to_session_slot_444` |  | no |
| 0x3109 | `virtual_no_args` |  | no |
| 0x3110 | `fun_0058d690` |  | no |
| 0xA101 | `fun_00592120` | LoginHandshake | no |
| 0xA102 | `fun_005921b0` | LoginRequest | no |
| 0xA104 | `fun_005922d0` |  | no |
| 0xA105 | `fun_005922a0` |  | no |
| 0xA107 | `virtual_no_args` |  | no |
| 0xA108 | `fun_00592380` |  | no |
| 0xA201 | `fun_00592220` | ServerList | no |
| 0xA202 | `fun_005923d0` | SelectServer | no |
| 0xA301 | `fun_00592420` | GameHandshake | no |
| 0xA302 | `fun_0058d790` |  | no |
| 0xA303 | `fun_0058d7e0` | Ping | no |
| 0xB101 | `fun_005969b0` | BankItemList | no |
| 0xB102 | `fun_00596a40` |  | no |
| 0xB103 | `fun_00596d70` |  | no |
| 0xB104 | `dispatch_packet_u32_to_session_slot_584` |  | no |
| 0xB105 | `dispatch_packet_u32_to_session_slot_588` |  | no |
| 0xB106 | `fun_00596e20` |  | no |
| 0xF101 | `fun_00593790` | ChatNormalAdmin | sí |
| 0xF102 | `fun_00593810` |  | no |
| 0xF103 | `fun_005938b0` | ChatWorldAdmin | sí |
| 0xF104 | `fun_00593940` |  | no |
| 0xF105 | `fun_005939d0` |  | no |
| 0xF106 | `fun_00593a50` |  | no |
| 0xF107 | `fun_0058d5f0` |  | no |
| 0xF109 | `fun_0058d640` |  | no |
| 0xF10A | `fun_00596310` |  | no |
| 0xF201 | `fun_00593ab0` |  | no |
| 0xF204 | `dispatch_packet_u16_to_session_slot_314` |  | no |
| 0xF501 | `dispatch_packet_u16_to_session_slot_318` | GmCommandError | no |
| 0xF601 | `dispatch_packet_u8_to_session_slot_3a0` |  | no |
| 0xF602 | `dispatch_packet_u8_to_session_slot_3a4` |  | no |
| 0xF603 | `fun_00594560` |  | no |
| 0xF701 | `fun_00594480` |  | no |
| 0xF709 | `virtual_no_args` |  | no |
| 0xF70A | `dispatch_packet_u16_to_session_slot_39c` |  | no |
| 0xF802 | `virtual_no_args` |  | no |
| 0xF803 | `virtual_no_args` |  | no |
| 0xF804 | `virtual_no_args` |  | no |
| 0xF805 | `dispatch_packet_u16_to_session_slot_380` |  | no |
| 0xF80B | `dispatch_packet_u16_to_session_slot_384` |  | no |
| 0xF80C | `dispatch_packet_u16_to_session_slot_388` |  | no |
| 0xF80F | `dispatch_packet_u16_to_session_slot_38c` |  | no |
| 0xF810 | `dispatch_packet_u16_to_session_slot_390` |  | no |
| 0xF901 | `fun_00593e30` |  | no |
| 0xF904 | `fun_00593fd0` |  | no |
| 0xF905 | `fun_00594050` |  | no |
| 0xF906 | `fun_00594150` |  | no |
| 0xF907 | `fun_005941b0` |  | no |
| 0xF908 | `fun_00594210` |  | no |
| 0xF909 | `fun_00594270` |  | no |
| 0xF90A | `fun_005942d0` |  | no |
| 0xF90B | `fun_00594330` |  | no |
| 0xF90C | `fun_005940d0` |  | no |
| 0xFA07 | `fun_00593be0` |  | no |
| 0xFA08 | `fun_00593c40` |  | no |
| 0xFA09 | `fun_00593cb0` |  | no |
| 0xFA0A | `fun_00593d30` |  | no |
| 0xFA0D | `virtual_no_args` |  | no |
| 0xFA0E | `virtual_no_args` |  | no |
| 0xFA0F | `virtual_no_args` |  | no |
| 0xFA10 | `virtual_no_args` |  | no |
| 0xFA11 | `fun_00593db0` |  | no |
| 0xFA12 | `fun_00593db0` |  | no |
