package channelserver

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type GuildMember struct {
	GuildID         uint32     `db:"guild_id"`
	CharID          uint32     `db:"character_id"`
	JoinedAt        *time.Time `db:"joined_at"`
	Souls           uint32     `db:"souls"`
	RPToday         uint16     `db:"rp_today"`
	RPYesterday     uint16     `db:"rp_yesterday"`
	Name            string     `db:"name"`
	IsApplicant     bool       `db:"is_applicant"`
	OrderIndex      uint8      `db:"order_index"`
	LastLogin       uint32     `db:"last_login"`
	Recruiter       bool       `db:"recruiter"`
	AvoidLeadership bool       `db:"avoid_leadership"`
	IsLeader        bool       `db:"is_leader"`
	HRP             uint16     `db:"hrp"`
	GR              uint16     `db:"gr"`
	WeaponID        uint16     `db:"weapon_id"`
	WeaponType      uint16     `db:"weapon_type"`
}

func (gm *GuildMember) CanRecruit() bool {
	if gm.Recruiter {
		return true
	}
	if gm.OrderIndex <= 3 {
		return true
	}
	if gm.IsLeader {
		return true
	}
	return false
}

func (gm *GuildMember) IsSubLeader() bool {
	return gm.OrderIndex <= 3
}

func (gm *GuildMember) Save(s *Session) error {
	_, err := s.server.db.Exec("UPDATE guild_characters SET avoid_leadership=$1, order_index=$2 WHERE character_id=$3", gm.AvoidLeadership, gm.OrderIndex, gm.CharID)

	if err != nil {
		s.logger.Error(
			"failed to update guild member data",
			zap.Error(err),
			zap.Uint32("charID", gm.CharID),
			zap.Uint32("guildID", gm.GuildID),
		)
		return err
	}
	return nil
}

const guildMembersSelectSQL = `
SELECT
	g.id as guild_id,
	joined_at,
	coalesce(souls, 0) as souls,
	COALESCE(rp_today, 0) AS rp_today,
	COALESCE(rp_yesterday, 0) AS rp_yesterday,
	c.name,
	character.character_id,
	coalesce(gc.order_index, 0) as order_index,
	c.last_login,
	coalesce(gc.recruiter, false) as recruiter,
	coalesce(gc.avoid_leadership, false) as avoid_leadership,
	c.hrp,
	c.gr,
	c.weapon_id,
	c.weapon_type,
	character.is_applicant,
	CASE WHEN g.leader_id = c.id THEN 1 ELSE 0 END as is_leader
	FROM (
		SELECT character_id, true as is_applicant, guild_id
		FROM guild_applications ga
		WHERE ga.application_type = 'applied'
		UNION
		SELECT character_id, false as is_applicant, guild_id
		FROM guild_characters gc
	) character
	JOIN characters c on character.character_id = c.id
	LEFT JOIN guild_characters gc ON gc.character_id = character.character_id
	JOIN guilds g ON g.id = character.guild_id
`

func GetGuildMembers(s *Session, guildID uint32, applicants bool) ([]*GuildMember, error) {
	rows, err := s.server.db.Queryx(fmt.Sprintf(`
			%s
			WHERE character.guild_id = $1 AND is_applicant = $2
	`, guildMembersSelectSQL), guildID, applicants)

	if err != nil {
		s.logger.Error("failed to retrieve membership data for guild", zap.Error(err), zap.Uint32("guildID", guildID))
		return nil, err
	}

	defer rows.Close()

	members := make([]*GuildMember, 0)

	for rows.Next() {
		member, err := buildGuildMemberObjectFromDBResult(rows, err, s)

		if err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return members, nil
}

func GetCharacterGuildData(s *Session, charID uint32) (*GuildMember, error) {
	rows, err := s.server.db.Queryx(fmt.Sprintf("%s	WHERE character.character_id=$1", guildMembersSelectSQL), charID)

	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to retrieve membership data for character '%d'", charID))
		return nil, err
	}

	defer rows.Close()

	hasRow := rows.Next()

	if !hasRow {
		return nil, nil
	}

	return buildGuildMemberObjectFromDBResult(rows, err, s)
}

func buildGuildMemberObjectFromDBResult(rows *sqlx.Rows, err error, s *Session) (*GuildMember, error) {
	memberData := &GuildMember{}

	err = rows.StructScan(&memberData)

	if err != nil {
		s.logger.Error("failed to retrieve guild data from database", zap.Error(err))
		return nil, err
	}

	return memberData, nil
}
