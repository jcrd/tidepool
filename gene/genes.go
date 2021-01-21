// This project is licensed under the MIT License (see LICENSE).

package gene

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

func (g Gene) String() string {
    switch g {
    case ZERO:
        return "0"
    case FWD:
        return ">"
    case BACK:
        return "<"
    case INC:
        return "+"
    case DEC:
        return "-"
    case READG:
        return "g"
    case WRITEG:
        return "G"
    case READB:
        return "b"
    case WRITEB:
        return "B"
    case LOOP:
        return "["
    case REP:
        return "]"
    case TURN:
        return "t"
    case XCHG:
        return "x"
    case KILL:
        return "k"
    case SHARE:
        return "s"
    case STOP:
        return "."
    default:
        return ""
    }
}
