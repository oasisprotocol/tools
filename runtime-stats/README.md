# Runtime stats

Collect various runtime stats.

## Build

```
go build
```

## Run

```
./runtime-stats <runtime-id> <start-height> <end-height> --address <path-to-node-internal.sock>  --genesis.file <path-to-genesis-json>

# Testnet cipher example.

./runtime-stats 0000000000000000000000000000000000000000000000000000000000000000 4695050 0 -a ./oasis-testnet/node/internal.sock  --genesis.file ./oasis-testnet/genesis.json

Runtime rounds: 472894
Successful rounds: 436169
Epoch transition rounds: 1157
Proposer timeouted rounds: 12154
Failed rounds: 23407
Discrepancies: 47757
Discrepancies (timeout): 70
Suspended: 7
Entity stats
|                  ENTITY ID                   | ELECTED | PRIMARY | BACKUP | PROPOSER | PRIMARY INVOKED | PRIMARY GOOD COMMIT | PRIM BAD COMMMIT | BCKP INVOKED | BCKP GOOD COMMIT | BCKP BAD COMMIT | PRIMARY MISSED | BCKP MISSED | PROPOSER MISSED | PROPOSED TIMEOUT |
|----------------------------------------------|---------|---------|--------|----------|-----------------|---------------------|------------------|--------------|------------------|-----------------|----------------|-------------|-----------------|------------------|
| pERaTagl7a57dXiVI3ZzoQbWskX3YmUHppeqcPdCVY8= |  104639 |   28571 |  84111 |    12198 |           27827 |               27825 |                0 |         1601 |              235 |               0 |              2 |        1366 |             150 |              251 |
| rL/+w2kRZEMSMn80TOooMa/UtyuDsI2zurI1u0ppoVY= |   14534 |    4906 |  11227 |     2168 |            4861 |                4856 |                0 |          239 |               51 |               0 |              5 |         188 |              21 |                8 |
| Efqu3z0lip+aDXkJUZqL9KfqpUIrgvISgGOdE/3OITw= |  137011 |   42989 | 105762 |    19010 |           41814 |               39127 |                0 |        11353 |             4427 |               0 |           2687 |        6926 |             606 |              245 |
| F/IE4bDGhg8naE7qk09uh+ZHP+UCuWTdesrTskzu0Ok= |   75995 |   27264 |  56430 |    11994 |           26035 |               24733 |                0 |         6136 |              779 |               0 |           1302 |        5357 |             774 |               74 |
| +G2csGBZ9P8Uf3Tvgfbuo3Z/1tFmqNMsiuSWen5s1qw= |   71185 |   20322 |  57603 |     9021 |           19548 |               19541 |                0 |         3488 |             2840 |               0 |              7 |         648 |             186 |              252 |
| np5ghSwh8QmA2CwtvgtbQ2p+BDspieZ5u9gu6pczHuo= |   18979 |    5944 |  14815 |     2577 |            5910 |                5910 |                0 |            0 |                0 |               0 |              0 |           0 |               4 |                5 |
| Y1FLFDa4o7JbRnjHc53AwvHzUEEShwEicxREwaKkPks= |   17573 |    7688 |  12843 |     3547 |            7670 |                7670 |                0 |            0 |                0 |               0 |              0 |           0 |               0 |                0 |
| nYK6gmglTXBNesG8vSFhh0V5b/BGubFCHz83XFgiQRM= |  134517 |   45825 | 103653 |    20926 |           44121 |               39039 |                0 |        10878 |              478 |               0 |           5082 |       10400 |             835 |              176 |
| W/Fj1JlO2hAgVSohQZZ3OQDb39Yf9O/jXxcOeW5E6AQ= |  120185 |   39223 |  90309 |    16644 |           37362 |               33666 |                0 |         9724 |             1072 |               0 |           3696 |        8652 |            1154 |              294 |
| /RupFkoz/Hep2Xm1eqtzwuLw+hS2AQNiKVgqQwnMalE= |   69577 |   24120 |  49268 |     9315 |           23183 |               23110 |                0 |          764 |              235 |               0 |             73 |         529 |             489 |              203 |
| 7C3hcBmlqELviUrYCTRsl0vY8GdwF5cC5ymBPVH8WSU= |  153091 |   56120 | 110946 |    23925 |           54755 |               53226 |                0 |        14159 |             6800 |               0 |           1529 |        7359 |             182 |              564 |
| eAktV5Y92L53zBVNuS+pVXjlwZ35BbGLjSXgS/niLzM= |  131819 |   43610 | 103711 |    19698 |           42048 |               40965 |                0 |         4333 |             1654 |               0 |           1083 |        2679 |             598 |              490 |
| yB1iGAzWLPEjRMN7Ll9ck+F0uEeHOgeQLyURzbPWb8k= |   57911 |   19194 |  43430 |     7966 |           18479 |               18479 |                0 |         1749 |             1445 |               0 |              0 |         304 |              67 |              382 |
| 3cn1uAKo0hyCq7gaUSAPIbFqchFJ5gAHOcyl8TcQGCw= |    2968 |    1788 |   1778 |      794 |            1784 |                1784 |                0 |            0 |                0 |               0 |              0 |           0 |               1 |                0 |
| 6Z6kefGqCQSNBvPd/whNez9YwS9QlLSKCmhWviZ1xg0= |   22629 |    5617 |  18613 |     2407 |            5451 |                5450 |                0 |            0 |                0 |               0 |              1 |           0 |             128 |                0 |
| PXxIgOYYSpJfpHEZBcRC1qLh7ZdILpk4npYWF0EQ75w= |  107512 |   39530 |  82817 |    18723 |           38076 |               36121 |                0 |         8454 |             2834 |               0 |           1955 |        5620 |             609 |              397 |
| L/xlgKf4Tx1XZogRBnuBmFS52NFoccW6QjMA8VS7g1g= |  167919 |   58588 | 125789 |    25495 |           57583 |               55307 |                0 |        10321 |             4485 |               0 |           2276 |        5836 |             206 |              328 |
| W51yg1gYW3yRwgnGYyGFGPW6YF/cnVoJYfALZNhWdhk= |  159132 |   46044 | 126623 |    19933 |           45263 |               44330 |                0 |         8764 |             5115 |               0 |            933 |        3649 |             158 |              335 |
| jDezUtFXRI6/x59WHVR49KlO7EWQZ29reqt043IyWFo= |  159285 |   54020 | 118850 |    23409 |           53020 |               49538 |                0 |        12678 |             1937 |               0 |           3482 |       10741 |              27 |              473 |
| bvUTh6r0m7zQSHutiMVq+V9HdxU2bCLWrlnsUi3RsBs= |  143165 |   46870 | 108228 |    20052 |           45844 |               43588 |                0 |        10339 |             3710 |               0 |           2256 |        6629 |             287 |              442 |
| t3jFBQQFkoVajtoje2EHB9oYiHYpoiaARtI2sag3+vU= |  161314 |   56191 | 115973 |    22355 |           55014 |               55009 |                0 |         8176 |             7436 |               0 |              5 |         740 |              72 |              533 |
| e4GM0u20JKAwvV7cUeix1J7JW1mLFiMetiMNzbDYm9U= |  149059 |   48778 | 115152 |    21814 |           47748 |               44537 |                0 |        12757 |             3758 |               0 |           3211 |        8999 |             385 |              272 |
| QTg+ZjubD/36egwByAIGC6lMVBKgqo7xnZPgHVoIKzc= |    8288 |    2514 |   6375 |     1556 |            2505 |                 562 |                0 |         4898 |                0 |               0 |           1943 |        4898 |               0 |                1 |
| 3xW+S+VeZ4atGmmxgfKfq0dDsIpe357CPrv56guaIng= |   57214 |   18594 |  43081 |     7688 |           18235 |               18234 |                0 |         1901 |             1371 |               0 |              1 |         530 |              63 |              124 |
| +lgliBmtfQ0ghj6SrEguOHr3cpS4a99Bs7iyD5pQOE0= |   61512 |   26014 |  43692 |    11394 |           25097 |               25090 |                0 |         1637 |             1477 |               0 |              7 |         160 |              72 |              508 |
| RKCvE04I/BWb7VC6tzAetR3cLSlmPOrshElcoFLCjnk= |  135835 |   46483 | 107100 |    21917 |           45381 |               43044 |                0 |        13101 |             2003 |               0 |           2337 |       11098 |             330 |              327 |
| LMByxWPjHKNOPuF5p0nigt7qf2P5VWA9OdnGYV6gJt0= |  163438 |   47360 | 126913 |    20230 |           45960 |               42771 |                0 |        13285 |             4118 |               0 |           3189 |        9167 |             502 |              279 |
| 4eM2SdjntkGGH0DqhloG8HJsPNSoEeYHFACuOV+9qbc= |  155292 |   50863 | 117812 |    22099 |           48152 |               46632 |                0 |        11883 |             2643 |               0 |           1520 |        9240 |              21 |             1994 |
| QgG1hFl4On84uVIrd++d3HxwoirdopH6oRmpltAIxxQ= |  163283 |   47021 | 126383 |    19528 |           45366 |               43213 |                0 |         9333 |             4000 |               0 |           2153 |        5333 |             944 |              148 |
| W4mvWjUL27JnIbC3iQOlrKdpsIVFoQ8RDpt6wv1mocw= |   66753 |   17815 |  51663 |     6845 |           17442 |               17358 |                0 |         1231 |              844 |               0 |             84 |         387 |             208 |               43 |
| 2xTkhlp0ISMMTKr7ksnOjj1for+Ruh/XMWDpw9zqOBY= |   59862 |   19150 |  43381 |     7277 |           18722 |               18721 |                0 |         1667 |             1393 |               0 |              1 |         274 |              53 |              173 |
| efBR83hww44OezMuvJ0T9Ewu4uPSQkOlO7EQq79KSis= |   35178 |   11970 |  26506 |     5086 |           11932 |               11932 |                0 |            0 |                0 |               0 |              0 |           0 |               0 |                3 |
| sKJ7WWqPBgo7Qg03SLb/lgO7J67BybQiFXn1cwMSFjU= |  163414 |   60788 | 119724 |    26555 |           59636 |               56893 |                0 |        13943 |             6613 |               0 |           2743 |        7330 |             129 |              401 |
| WBgzYbX1ks0zB33f92JcBXdb5kVam7Sx1G7uepugqBI= |  148303 |   40007 | 122169 |    18793 |           38512 |               25051 |                0 |        13983 |             1233 |               0 |          13461 |       12750 |             889 |              189 |
| ucBShpGap3zSeKqkAo5XRcfeOZrvQYAK37CKUHrCK9A= |  152253 |   39370 | 122949 |    17219 |           38369 |               26422 |                0 |        12777 |             1513 |               0 |          11947 |       11264 |             201 |              302 |
| LXXBwwqBgwP3Jd3qCAWDpDBYmeMvX0mtmCsCUveGKuk= |  171029 |   59811 | 128055 |    25977 |           58387 |               56729 |                0 |        13361 |             7313 |               0 |           1658 |        6048 |             456 |              466 |
| f5xc3Yk/mzo3aBFaZMp80PlUjnmCkP5aj/I8i79Y/Y0= |     104 |       0 |    104 |        0 |               0 |                   0 |                0 |            2 |                0 |               0 |              0 |           2 |               0 |                0 |
| gSUCphbQ9ZcWncr3NCjKuGfhJ27YjMtLGFgCCp3RvFA= |   95411 |   27782 |  75951 |    12040 |           27096 |               27096 |                0 |         2943 |             2013 |               0 |              0 |         930 |             132 |              239 |
| dyh4HqSlTabRSgqUHmcUOUt2bmcjCkmTDL3IAZV9ZCc= |  142110 |   43831 | 109234 |    18793 |           42176 |               39738 |                0 |        13218 |              637 |               0 |           2438 |       12581 |             856 |              282 |
| ZdA1Wc50Pe3eWSUcmiznyOwApRTSHg3xFgjSHuv1Zjo= |  147190 |   46879 | 112197 |    19961 |           46096 |               43896 |                0 |        10832 |             4409 |               0 |           2200 |        6423 |             112 |              395 |
| w+RegHqNnDGjEHukprzt4CZGr24ZJ8NY7scX7yuK2R4= |  146884 |   57944 | 106463 |    25792 |           56725 |               53434 |                0 |        11501 |             5225 |               0 |           3291 |        6276 |             247 |              556 |
```
