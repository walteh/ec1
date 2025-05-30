package oci_image_cache

 type OCICachedImage string


var Registry = map[OCICachedImage][]byte{}

func (me OCICachedImage) String() string {
	return string(me)
}

