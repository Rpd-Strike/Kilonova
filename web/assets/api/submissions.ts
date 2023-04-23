import { getCall } from "../net";

export type Submission = {
	id: number;
	created_at: string;
	user_id: number;
	problem_id: number;
	language: string;
	code?: string;
	code_size: number;
	compile_error: boolean;
	compile_message?: string;
	contest_id: number | null;
	max_time: number;
	max_memory: number;
	score: number;
	status: string;
};

export type UserBrief = {
	id: number;
	name: string;
	admin: boolean;
	proposer: boolean;
	bio?: string;
};

export type Problem = {
	id: number;
	created_at: string;
	name: string;
	test_name: string;
	default_points: number;
	visible: boolean;
	time_limit: number;
	memory_limit: number;
	source_credits: string;
	author_credits: string;
	console_input: boolean;
	scoring_strategy: "sum_subtasks" | "max_submission";
};

export type ResultSubmission = {
	sub: Submission;
	author: UserBrief;
	problem: Problem;
};

export type SubmissionQuery = {
	user_id?: number;
	problem_id?: number;
	contest_id?: number;
	score?: number;
	status?: string;
	lang?: string;

	page: number;

	compile_error?: boolean;
	ordering?: string;
	ascending?: boolean;

	limit?: number;
};

function serializeQuery(q: SubmissionQuery): object {
	return {
		ordering: typeof q.ordering !== "undefined" ? q.ordering : "id",
		ascending: (typeof q.ordering !== "undefined" && q.ascending) || false,
		user_id: typeof q.user_id !== "undefined" && q.user_id > 0 ? q.user_id : undefined,
		problem_id: typeof q.problem_id !== "undefined" && q.problem_id > 0 ? q.problem_id : undefined,
		contest_id: typeof q.contest_id !== "undefined" && q.contest_id > 0 ? q.contest_id : undefined,
		status: q.status !== "" ? q.status : undefined,
		score: typeof q.score !== "undefined" && q.score >= 0 ? q.score : undefined,
		lang: typeof q.lang !== "undefined" && q.lang !== "" ? q.lang : undefined,
		compile_error: q.compile_error,
		offset: (q.page - 1) * 50,
		limit: typeof q.limit !== "undefined" && q.limit > 0 ? q.limit : 50,
	};
}

export async function getSubmissions(q: SubmissionQuery) {
	let res = await getCall<{
		count: number;
		subs: ResultSubmission[];
	}>("/submissions/get", serializeQuery(q));
	if (res.status === "error") {
		throw new Error(res.data);
	}
	return res.data;
}

export async function getUser(uid: number) {
	let res = await getCall<UserBrief>("/user/get", { id: uid });
	if (res.status === "error") {
		throw new Error(res.data);
	}
	return res.data;
}

export type SubTest = {
	id: number;
	created_at: string;
	done: boolean;
	verdict: string;
	time: number;
	memory: number;
	score: number;
	test_id: number;
	user_id: number;
	submission_id: number;

	visible_id: number;
	max_score: number;
};

export type SubmissionSubTask = {
	id: number;
	created_at: string;

	submission_id: number;
	user_id: number;
	subtask_id?: number;

	problem_id: number;
	visible_id: number;
	score: number;
	final_percentage?: number;

	subtests: number[];
};

export type FullSubmission = Submission & {
	author: UserBrief;
	problem: Problem;
	subtests: SubTest[];
	subtasks: SubmissionSubTask[];

	problem_editor: boolean;
};

export async function getSubmission(subID: number) {
	let res = await getCall<FullSubmission>("/submissions/getByID", { id: subID });
	if (res.status === "error") {
		throw new Error(res.data);
	}
	return res.data;
}
