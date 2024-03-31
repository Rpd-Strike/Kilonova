package sudoapi

import (
	"context"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/integrations/moss"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var (
	NormalUserVirtualContests = config.GenFlag[bool]("behavior.contests.anyone_virtual", false, "Anyone can create virtual contests")
	NormalUserVCLimit         = config.GenFlag[int]("behavior.contests.normal_user_max_day", 10, "Number of maximum contests a non-proposer can create per day")
)

func (s *BaseAPI) CreateContest(ctx context.Context, name string, cType kilonova.ContestType, author *UserBrief) (int, *StatusError) {
	if author == nil {
		return -1, ErrMissingRequired
	}
	if !(cType == kilonova.ContestTypeNone || cType == kilonova.ContestTypeOfficial || cType == kilonova.ContestTypeVirtual) {
		return -1, Statusf(400, "Invalid contest type")
	}
	if cType == kilonova.ContestTypeNone {
		cType = kilonova.ContestTypeVirtual
	}
	if cType == kilonova.ContestTypeOfficial && !author.IsAdmin() {
		cType = kilonova.ContestTypeVirtual
	}

	if !author.IsProposer() {
		if !NormalUserVirtualContests.Value() {
			return -1, Statusf(403, "Creation of contests by non-proposers has been disabled")
		}

		// Enforce stricter limits for non-proposers
		since := time.Now().Add(-24 * time.Hour) // rolling day
		cnt, err := s.db.ContestCount(ctx, kilonova.ContestFilter{
			Since:    &since,
			EditorID: &author.ID,
		})
		if err != nil || (cnt >= NormalUserVCLimit.Value() && NormalUserVCLimit.Value() >= 0) {
			if err != nil {
				zap.S().Warn(err)
			}
			return -1, Statusf(400, "You can create at most %d contests per day", NormalUserVCLimit.Value())
		}
	}
	id, err := s.db.CreateContest(ctx, name, cType)
	if err != nil {
		return -1, WrapError(err, "Couldn't create contest")
	}
	if err := s.db.AddContestEditor(ctx, id, author.ID); err != nil {
		zap.S().Warn(err)
		return id, WrapError(err, "Couldn't add author to contest editors")
	}
	return id, nil
}

func (s *BaseAPI) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) *kilonova.StatusError {
	if err := s.db.UpdateContest(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest")
	}
	return nil
}

func (s *BaseAPI) UpdateContestProblems(ctx context.Context, id int, list []int) *StatusError {
	if err := s.db.UpdateContestProblems(ctx, id, list); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest problems")
	}
	return nil
}

func (s *BaseAPI) DeleteContest(ctx context.Context, contest *kilonova.Contest) *StatusError {
	if contest == nil {
		return Statusf(400, "Invalid contest")
	}
	if err := s.db.DeleteContest(ctx, contest.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete contest")
	}
	s.LogUserAction(ctx, "Removed contest #%d: %q", contest.ID, contest.Name)
	return nil
}

func (s *BaseAPI) Contest(ctx context.Context, id int) (*kilonova.Contest, *StatusError) {
	contest, err := s.db.Contest(ctx, id)
	if err != nil || contest == nil {
		return nil, WrapError(ErrNotFound, "Contest not found")
	}
	return contest, nil
}

func (s *BaseAPI) Contests(ctx context.Context, filter kilonova.ContestFilter) ([]*kilonova.Contest, *StatusError) {
	contests, err := s.db.Contests(ctx, filter)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) ContestCount(ctx context.Context, filter kilonova.ContestFilter) (int, *StatusError) {
	cnt, err := s.db.ContestCount(ctx, filter)
	if err != nil {
		return -1, WrapError(err, "Couldn't fetch contests")
	}
	return cnt, nil
}

func (s *BaseAPI) VisibleFutureContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	filter := kilonova.ContestFilter{
		Future:      true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
	}
	var uid = -1
	if user != nil {
		uid = user.ID
	}
	filter.ImportantContestsUID = &uid
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) VisibleRunningContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	filter := kilonova.ContestFilter{
		Running:     true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
		Ordering:    "end_time",
	}
	var uid = -1
	if user != nil {
		uid = user.ID
	}
	filter.ImportantContestsUID = &uid
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) ProblemRunningContests(ctx context.Context, problemID int) ([]*kilonova.Contest, *StatusError) {
	return s.Contests(ctx, kilonova.ContestFilter{
		Running:   true,
		ProblemID: &problemID,
		Ascending: true,
		Ordering:  "end_time",
	})
}

func (s *BaseAPI) ContestLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time, filter kilonova.UserFilter) (*kilonova.ContestLeaderboard, *StatusError) {
	switch contest.LeaderboardStyle {
	case kilonova.LeaderboardTypeClassic:
		leaderboard, err := s.db.ContestClassicLeaderboard(ctx, contest, freezeTime, &filter)
		if err != nil {
			return nil, WrapError(err, "Couldn't generate leaderboard")
		}
		return leaderboard, nil
	case kilonova.LeaderboardTypeICPC:
		leaderboard, err := s.db.ContestICPCLeaderboard(ctx, contest, freezeTime, &filter)
		if err != nil {
			return nil, WrapError(err, "Couldn't generate leaderboard")
		}
		return leaderboard, nil
	default:
		return nil, Statusf(400, "Invalid contest leaderboard type")
	}
}

func (s *BaseAPI) CanJoinContest(c *kilonova.Contest) bool {
	if !c.PublicJoin {
		return false
	}
	if c.RegisterDuringContest && !c.Ended() { // Registration during contest is enabled
		return true
	}
	return !c.Started()
}

// CanSubmitInContest checks if the user is either a contestant and the contest is running, or a tester/editor/admin.
// Ended contests cannot have submissions created by anyone
// Also, USACO-style contests are fun to handle...
func (s *BaseAPI) CanSubmitInContest(user *kilonova.UserBrief, c *kilonova.Contest) bool {
	if c.Ended() {
		return false
	}
	if s.IsContestTester(user, c) {
		return true
	}
	if user == nil || c == nil {
		return false
	}
	if !c.Running() {
		return false
	}
	reg, err := s.db.ContestRegistration(context.Background(), c.ID, user.ID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}

	if reg == nil {
		return false
	}

	if c.PerUserTime == 0 { // Normal, non-USACO contest, only registration matters
		return true
	}

	// USACO contests are a bit more finnicky
	if reg.IndividualStartTime == nil || reg.IndividualEndTime == nil { // Hasn't pressed start yet
		return false
	}
	return time.Now().After(*reg.IndividualStartTime) && time.Now().Before(*reg.IndividualEndTime) // During window of visibility
}

// CanViewContestProblems checks if the user can see a contest's problems.
// Note that this does not neccesairly mean that he can submit in them!
// A problem may be viewable because the contest is running and visible, but only registered people should submit
// It's a bit frustrating but it's an important distinction
// If you think about it, all submitters can view problems, but not all problem viewers can submit
func (s *BaseAPI) CanViewContestProblems(ctx context.Context, user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if s.IsContestTester(user, contest) { // Tester + Editor + Admin
		return true
	}
	if !contest.Started() {
		return false
	}
	if contest.Ended() && contest.Visible { // Once ended and visible, it's free for all
		return true
	}
	if contest.PerUserTime == 0 && contest.Visible && !contest.RegisterDuringContest {
		// Problems can be seen by anyone only on visible, non-USACO contests that disallow registering during contest
		return true
	}
	return s.CanSubmitInContest(user, contest)
}

func (s *BaseAPI) CanViewContestLeaderboard(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !s.IsContestVisible(user, contest) {
		return false
	}
	if s.IsContestTester(user, contest) { // Tester + Editor + Admin
		return true
	}
	// Otherwise, normal contestant
	if !contest.Started() {
		// Non-started contests can leak problem IDs/names
		return false
	}
	return contest.PublicLeaderboard
}

func (s *BaseAPI) AddContestEditor(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest editor: sanity strip failed")
	}
	if err := s.db.AddContestEditor(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest editor")
	}
	return nil
}

func (s *BaseAPI) AddContestTester(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest tester: sanity strip failed")
	}
	if err := s.db.AddContestTester(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest tester")
	}
	return nil
}

func (s *BaseAPI) StripContestAccess(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't strip contest access")
	}
	return nil
}

func (s *BaseAPI) RunMOSS(ctx context.Context, contest *kilonova.Contest) *StatusError {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{ContestID: &contest.ID})
	if err != nil {
		return err
	}

	for _, pb := range pbs {
		subs, err := s.RawSubmissions(ctx, kilonova.SubmissionFilter{
			ProblemID: &pb.ID,
			ContestID: &contest.ID,

			Ordering:  "score",
			Ascending: false,
		})
		if err != nil {
			return err
		}
		if len(subs) == 0 {
			continue
		}
		mossSubs := make(map[string][]*kilonova.Submission)
		for _, sub := range subs {
			name := eval.Langs[sub.Language].MOSSName
			// TODO: See if this can be simplified?
			_, ok := mossSubs[name]
			if !ok {
				mossSubs[name] = []*kilonova.Submission{sub}
			} else {
				mossSubs[name] = append(mossSubs[name], sub)
			}
		}

		for mossLang, subs := range mossSubs {
			var lang eval.Language
			for _, elang := range eval.Langs {
				if elang.MOSSName == mossLang && (lang.InternalName == "" || lang.InternalName < elang.InternalName) {
					lang = elang
				}
			}

			zap.S().Debugf("%s - %s - %d", pb.Name, lang.InternalName, len(subs))
			conn, err1 := moss.New(ctx)
			if err1 != nil {
				return WrapError(err, "Could not initialize MOSS")
			}
			users := make(map[int]bool)
			for _, sub := range subs {
				if _, ok := users[sub.UserID]; ok {
					continue
				}
				user, err := s.UserBrief(ctx, sub.UserID)
				if err != nil {
					return err
				}
				users[sub.UserID] = true

				code, err := s.RawSubmissionCode(ctx, sub.ID)
				if err != nil {
					return err
				}
				conn.AddFile(eval.Langs[sub.Language], user.Name, code)
			}
			url, err1 := conn.Process(&moss.Options{
				Language: lang,
				Comment:  fmt.Sprintf("%s - %s (%s)", contest.Name, pb.Name, lang.PrintableName),
			})
			if err1 != nil {
				return WrapError(err, "Could not get MOSS result")
			}
			if _, err := s.db.InsertMossSubmission(ctx, contest.ID, pb.ID, lang, url, len(users)); err != nil {
				return WrapError(err, "Could not commit MOSS result to DB")
			}
		}
	}
	return nil
}

func (s *BaseAPI) MOSSSubmissions(ctx context.Context, contestID int) ([]*kilonova.MOSSSubmission, *StatusError) {
	subs, err := s.db.MossSubmissions(ctx, contestID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch MOSS submissions")
	}
	return subs, nil
}
