|       FOR stmt2 ';' expr ';' stmt2 block
        {
                block := append($7, $6)
                loop := $1.NewFun($4, block.NewFun())
                $$ = list{$2, loop}
        }
