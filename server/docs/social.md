# Native social service

The EP5 social sheet includes Guild, Buddy, Block and Search Party pages.
GX Server now supplies the existing 0x2201–0x220e social wire family, rather than
an AzerothCore extension. Native reader/writer references include 0x595140,
0x5953a0 and 0x59bb30–0x59bd80.

Buddy invitations require an online character of the same faction on the same
map. Acceptance stores both directions in PS_GameData_FriendChars in one SQLite
transaction. Removal affects the requesting character's list. Notes are stored
in the existing Memo column. Each list is limited to 50 entries.

Block uses PS_GameData_BanChars independently of Buddy membership and rejects
incoming whispers from blocked characters. The Go client also filters blocked
names/IDs across its displayed chat channels. Presence snapshots refresh every
three seconds; list membership and notes always follow server responses.

Search Party registers the requester and returns registered online candidates
of the same faction who are outside a party. The UI can whisper or invite a
selected candidate through the existing party transport. Registrations and
pending invitations are session state and are cleared on disconnect. This is
GX Server's implemented candidate policy; complete comparison of the original
service's filtering and exceptional conditions remains ongoing.

Native result meanings are retained: Buddy 0 rejected, 1 accepted, 2 unavailable
in the same area (SysMsg 399–401); Block success/failure uses 402/403; Search Party
registration uses 404/405. The client does not predict membership before an ACK.

The service uses existing tables and needs no schema migration. Portable builds
include it. An already running world process needs its next restart to load a
new executable; updating files never replaces the live database.
