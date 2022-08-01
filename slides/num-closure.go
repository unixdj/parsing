	case 15:
		yyDollar = yyS[yypt-1 : yypt+1]
		{
			yyVAL.fun = func() int {
				return yyDollar[1].num
			}
		}
