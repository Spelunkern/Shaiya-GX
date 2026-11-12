package proto

// GuildInfo is the 120-byte entry in 0x0D2F and 0x0D30.
type GuildInfo struct {
	ID                   uint32
	Name, Master, Remark string
	Rank                 uint8
	Points               uint32
}

type GuildMember struct {
	ID     uint32
	Rank   uint8
	Level  uint16
	Class  uint8
	Name   string
	Online bool
}

type GuildCandidate struct {
	ID    uint32
	Level uint16
	Class uint8
	Name  string
}

func readGuildApplicant(r *Reader) GuildCandidate {
	return GuildCandidate{ID: r.U32(), Level: r.U16(), Class: r.U8(), Name: r.Fixed(21)}
}

func DecodeGuildApplicants(p []byte) ([]GuildCandidate, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/28 {
		return nil, ErrShort
	}
	list := make([]GuildCandidate, n)
	for i := range list {
		list[i] = readGuildApplicant(r)
	}
	return list, r.Err()
}

func DecodeGuildApplicant(p []byte) (GuildCandidate, error) {
	r := NewReader(p)
	v := readGuildApplicant(r)
	return v, r.Err()
}

type GuildJoin struct {
	Result uint8
	ID     uint32
	Rank   uint8
	Name   string
}

type GuildCreateVote struct {
	Master       uint32
	Name, Remark string
}

func DecodeGuildCreateVote(p []byte) (GuildCreateVote, error) {
	r := NewReader(p)
	v := GuildCreateVote{Master: r.U32(), Name: r.Fixed(25), Remark: r.Fixed(65)}
	return v, r.Err()
}

func DecodeGuildCreateFailure(p []byte) (uint8, error) {
	r := NewReader(p)
	result := r.U8()
	r.Bytes(95)
	return result, r.Err()
}

func readGuildInfo(r *Reader) GuildInfo {
	return GuildInfo{ID: r.U32(), Name: r.Fixed(25), Master: r.Fixed(21), Remark: r.Fixed(65), Rank: r.U8(), Points: r.U32()}
}

func DecodeGuildList(p []byte) ([]GuildInfo, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/120 {
		return nil, ErrShort
	}
	list := make([]GuildInfo, n)
	for i := range list {
		list[i] = readGuildInfo(r)
	}
	return list, r.Err()
}

func DecodeGuildInfo(p []byte) (GuildInfo, error) {
	r := NewReader(p)
	v := readGuildInfo(r)
	return v, r.Err()
}

func readGuildMember(r *Reader, online bool) GuildMember {
	return GuildMember{ID: r.U32(), Rank: r.U8(), Level: r.U16(), Class: r.U8(), Name: r.Fixed(21), Online: online}
}

func DecodeGuildMembers(p []byte, online bool) ([]GuildMember, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/29 {
		return nil, ErrShort
	}
	list := make([]GuildMember, n)
	for i := range list {
		list[i] = readGuildMember(r, online)
	}
	return list, r.Err()
}

func DecodeGuildMemberAdded(p []byte) (GuildMember, error) {
	r := NewReader(p)
	online := r.U8() != 0
	m := readGuildMember(r, online)
	return m, r.Err()
}

func DecodeGuildJoin(p []byte) (GuildJoin, error) {
	r := NewReader(p)
	v := GuildJoin{Result: r.U8(), ID: r.U32(), Rank: r.U8(), Name: r.Fixed(25)}
	return v, r.Err()
}

func DecodeGuildCreate(p []byte) (GuildJoin, string, error) {
	r := NewReader(p)
	v := GuildJoin{Result: r.U8(), ID: r.U32(), Rank: r.U8(), Name: r.Fixed(25)}
	remark := r.Fixed(65)
	return v, remark, r.Err()
}

func GuildInvitePacket(id uint32) []byte { return idPacket(GuildInvite, id) }
func GuildAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildInviteAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func GuildLeavePacket() []byte   { return NewWriter(GuildLeave).Bytes() }
func GuildDisbandPacket() []byte { return NewWriter(GuildDisband).Bytes() }
func GuildDisbandAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildDisbandAgree)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func GuildExpelPacket(id uint32) []byte { return idPacket(GuildExpel, id) }
func GuildApplyPacket(id uint32) []byte { return idPacket(GuildApply, id) }
func GuildCreatePacket(name, remark string) []byte {
	w := NewWriter(GuildCreate)
	w.Fixed(name, 25, true)
	w.Fixed(remark, 65, true)
	return w.Bytes()
}
func GuildCreateAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildCreateAgree)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func GuildApplicantAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GuildJoinResult)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(id)
	return w.Bytes()
}
func GuildRankPacket(id uint32, demote bool) []byte {
	w := NewWriter(GuildStatus)
	if demote {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(id)
	return w.Bytes()
}
