package models

func (b *Bean) LikeTargetID() string           { return b.Id }
func (b *Bean) LikeTargetType() LikeTargetType { return LikeTargetTypeBean }
func (b *Bean) ApplyLikeInfo(isLiked bool, count int) {
	b.IsLiked = isLiked
	b.LikesCount = int32(count)
}
