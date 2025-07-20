// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package rate

import "time"

type Limiter struct {
	limit float64
	rate  float64

	lastRefill time.Time
	tokens     float64
}

// rate refills limiter rate tokens per second
func NewLimiter(rate float64, limit float64) Limiter {
	return Limiter{
		limit:      limit,
		rate:       rate,
		lastRefill: time.Now().UTC(),
		tokens:     limit,
	}
}

func (rl *Limiter) Fill() {
	rl.update()
	rl.tokens = rl.limit
}

func (rl *Limiter) SetRate(rate float64) {
	rl.update()
	rl.rate = rate
}

func (rl *Limiter) SetLimit(limit float64) {
	rl.update()
	rl.limit = limit
}

func (rl *Limiter) Take(tokens float64) bool {
	rl.update()
	has := rl.Has(tokens)
	if has {
		rl.tokens -= tokens
		return true
	}
	return false
}

func (rl *Limiter) Has(tokens float64) bool {
	rl.update()
	return rl.tokens >= tokens
}

func (rl *Limiter) update() {
	lastRefill := rl.lastRefill
	rl.lastRefill = time.Now().UTC()

	rl.tokens += min(time.Since(lastRefill).Seconds()*rl.rate, rl.limit)
}
