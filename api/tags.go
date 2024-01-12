package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) getTags(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Type kilonova.TagType `json:"type"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Type == kilonova.TagTypeNone {
		tags, err := s.base.Tags(r.Context())
		if err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, tags)
		return
	}

	if !kilonova.ValidTagType(args.Type) {
		errorData(w, "Invalid tag type", 400)
		return
	}

	tags, err := s.base.TagsByType(r.Context(), args.Type)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, tags)
}

func (s *API) createTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name string           `json:"name"`
		Type kilonova.TagType `json:"type"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Type == kilonova.TagTypeNone {
		args.Type = kilonova.TagTypeOther
	}

	if !kilonova.ValidTagType(args.Type) {
		errorData(w, "Invalid tag type", 400)
		return
	}

	id, err := s.base.CreateTag(r.Context(), args.Name, args.Type)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

func (s *API) updateTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID int `json:"id"`

		Type kilonova.TagType `json:"type"`
		Name *string          `json:"name"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	tag, err := s.base.TagByID(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if args.Type != kilonova.TagTypeNone && args.Type != tag.Type {
		if err := s.base.UpdateTagType(r.Context(), tag, args.Type); err != nil {
			err.WriteError(w)
			return
		}
	}

	if args.Name != nil && *args.Name != tag.Name {
		if err := s.base.UpdateTagName(r.Context(), tag, *args.Name); err != nil {
			err.WriteError(w)
			return
		}
	}

	returnData(w, "Updated tag")
}

func (s *API) updateProblemTags(ctx context.Context, args struct {
	Tags []int `json:"tags"`
}) *kilonova.StatusError {
	return s.base.UpdateProblemTags(ctx, util.ProblemContext(ctx).ID, args.Tags)
}

func (s *API) problemTags(ctx context.Context, args struct{}) ([]*kilonova.Tag, *kilonova.StatusError) {
	return s.base.ProblemTags(ctx, util.ProblemContext(ctx).ID)
}
