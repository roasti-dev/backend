package models

func (p *Article) LikeTargetID() string           { return p.Id }
func (p *Article) LikeTargetType() LikeTargetType { return LikeTargetTypeArticle }
func (p *Article) ApplyLikeInfo(isLiked bool, count int) {
	p.IsLiked = isLiked
	p.LikesCount = int32(count)
}
