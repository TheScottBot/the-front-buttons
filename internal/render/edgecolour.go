package render

import (
	"image"
	"image/color"
)

// edgeOpacityMinimum is how opaque a pixel must be to count as the icon's
// edge. A macOS icon is a rounded square in a transparent margin with a soft
// drop shadow; the shadow is faint, so a high threshold skips it and lands on
// the shape itself.
const edgeOpacityMinimum = 0xe6

// edgeBucketShift groups edge colours into buckets of 32 levels per channel, so
// the slight shading and anti aliasing along an edge count as one colour and a
// few stray pixels cannot outvote it.
const edgeBucketShift = 5

// edgeBucket is one group of similar edge colours: how many pixels fell in it,
// and their channel totals, so the colour reported is their mean rather than
// the bucket's corner.
type edgeBucket struct {
	count            int
	red, green, blue int
	firstSeen        int
}

// EdgeColour returns the colour of an icon's visible edge: the first opaque
// pixel met scanning inward from each side, along every row and column, and
// the colour most of those pixels share. A plain average would mix in the
// transparent margin and whatever is drawn in the middle; taking the dominant
// colour of the outline gives the colour a person sees as the icon's border.
// It reports false when nothing in the image is opaque enough to be an edge.
func EdgeColour(icon image.Image) (color.RGBA, bool) {
	buckets := map[int]*edgeBucket{}
	seen := 0

	record := func(x int, y int) bool {
		pixel, _ := color.NRGBAModel.Convert(icon.At(x, y)).(color.NRGBA)
		if pixel.A < edgeOpacityMinimum {
			return false
		}

		key := int(pixel.R>>edgeBucketShift)<<redShift | int(pixel.G>>edgeBucketShift)<<greenShift |
			int(pixel.B>>edgeBucketShift)

		bucket, exists := buckets[key]
		if !exists {
			bucket = &edgeBucket{count: 0, red: 0, green: 0, blue: 0, firstSeen: seen}
			buckets[key] = bucket
		}

		bucket.count++
		bucket.red += int(pixel.R)
		bucket.green += int(pixel.G)
		bucket.blue += int(pixel.B)
		seen++

		return true
	}

	scanEdges(icon.Bounds(), record)

	return dominantColour(buckets)
}

// scanEdges walks inward from each side of the bounds, along every row and
// every column, stopping each walk at the first pixel record accepts.
func scanEdges(bounds image.Rectangle, record func(x int, y int) bool) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		scanFromBothEnds(bounds.Min.X, bounds.Max.X, func(x int) bool { return record(x, y) })
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		scanFromBothEnds(bounds.Min.Y, bounds.Max.Y, func(y int) bool { return record(x, y) })
	}
}

// scanFromBothEnds offers positions from start upward until one is accepted,
// then from just before end downward until one is accepted.
func scanFromBothEnds(start int, end int, accept func(position int) bool) {
	for position := start; position < end; position++ {
		if accept(position) {
			break
		}
	}

	for position := end - 1; position >= start; position-- {
		if accept(position) {
			break
		}
	}
}

// dominantColour returns the mean colour of the fullest bucket. A tie goes to
// the bucket seen first, so the same icon always gives the same colour.
func dominantColour(buckets map[int]*edgeBucket) (color.RGBA, bool) {
	var fullest *edgeBucket

	for _, bucket := range buckets {
		if fullest == nil || bucket.count > fullest.count ||
			(bucket.count == fullest.count && bucket.firstSeen < fullest.firstSeen) {
			fullest = bucket
		}
	}

	if fullest == nil {
		return color.RGBA{R: 0, G: 0, B: 0, A: 0}, false
	}

	return color.RGBA{
		R: channelMean(fullest.red, fullest.count),
		G: channelMean(fullest.green, fullest.count),
		B: channelMean(fullest.blue, fullest.count),
		A: maximumChannelValue,
	}, true
}

// channelMean divides a channel's total by its pixel count. Every value added
// was a channel, so the mean is one too; the check only proves it.
func channelMean(total int, count int) uint8 {
	mean := total / count
	if mean < 0 || mean > maximumChannelValue {
		return maximumChannelValue
	}

	return uint8(mean)
}
