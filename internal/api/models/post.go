package models

func (p *Post) LikeTargetID() string           { return p.Id }
func (p *Post) LikeTargetType() LikeTargetType { return LikeTargetTypePost }
func (p *Post) ApplyLikeInfo(isLiked bool, count int) {
	p.IsLiked = isLiked
	p.LikesCount = int32(count)
}
