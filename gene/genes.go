// This project is licensed under the MIT License (see LICENSE).

package gene

import (
    "encoding/json"
)

type Gene int
type Genome []Gene

const (
    ZERO Gene = iota
    FWD
    BACK
    INC
    DEC
    READG
    WRITEG
    READB
    WRITEB
    LOOP
    REP
    TURN
    XCHG
    KILL
    SHARE
    STOP

    N
)

var geneChars = map[Gene]string{
    ZERO: "0",
    FWD: "}",
    BACK: "{",
    INC: "+",
    DEC: "-",
    READG: "g",
    WRITEG: "G",
    READB: "b",
    WRITEB: "B",
    LOOP: "[",
    REP: "]",
    TURN: "t",
    XCHG: "x",
    KILL: "k",
    SHARE: "s",
    STOP: ".",
}

func (g Gene) String() string {
    return geneChars[g]
}

func (g Genome) String() string {
    var s string
    for _, gene := range g {
        s += gene.String()
    }
    return s
}

func (g Genome) MarshalJSON() ([]byte, error) {
    return json.Marshal(g.String())
}
