package motherboard

type Tile [16]uint8

type PaletteTile [8][8]uint8

func (t *Tile) ParseTile() PaletteTile {
	var tile PaletteTile
	for i := 0; i < 15; i++ {
		for j := 0; j < 8; j++ {
			tile[i>>1][j] = (((t[i+1] >> j) & 0x1) << 1) | ((t[i] >> j) & 0x1)
		}
		i++
	}
	return tile
}
