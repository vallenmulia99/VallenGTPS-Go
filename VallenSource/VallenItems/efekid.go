package items

// GetPunchEffect mengembalikan efek punch ID berdasarkan item ID yang di-equip di hand slot.
// Mapping ini diambil dari fungsi get_punch_id() referensi GrowTavern/BaseServer.h.
// Return 0 jika item tidak punya efek punch khusus (default fist).
func GetPunchEffect(itemID int) int {
	switch itemID {
	case 138, 2976, 8354:
		return 1
	case 14564, 14566, 14568, 14570, 14572, 14574, 14576, 14220, 14222, 14598,
		19988, 19990, 19958, 19960, 19962:
		return 237
	case 5924:
		return 30
	case 366, 1464:
		return 2
	case 472:
		return 3
	case 594, 10130, 5424, 5456, 4136, 10052:
		return 4
	case 768:
		return 5
	case 900, 7760, 9272, 5002, 7758, 4664, 8046, 9062, 1252, 1254, 9086,
		3680, 5176, 7750, 1228, 3430:
		return 6
	case 910, 4332, 1250, 12656:
		return 7
	case 930, 1010, 6382, 9698, 6368:
		return 8
	case 1016, 6058:
		return 9
	case 1204, 9534, 10928:
		return 10
	case 1378:
		return 11
	case 1440, 4508, 2224, 4512, 4510, 4515, 10996, 9654, 4514, 11764, 7748:
		return 12
	case 1484, 5160, 9802, 9508:
		return 13
	case 1512, 1648:
		return 14
	case 1542:
		return 15
	case 1576:
		return 16
	case 1676, 7504:
		return 17
	case 1710, 4644, 1714, 1712, 6044, 1570:
		return 18
	case 1748, 8006, 8008, 8010, 8012:
		return 19
	case 3578:
		return 19
	case 1780:
		return 20
	case 1782, 5156, 9776, 9810, 10120:
		return 21
	case 1804, 5194, 9784:
		return 22
	case 1868, 7754, 1998:
		return 23
	case 1874:
		return 24
	case 1946, 2800:
		return 25
	case 1952, 2854, 9280, 1974:
		return 26
	case 1956:
		return 27
	case 1960:
		return 28
	case 2908, 6312, 9496, 8554, 3162, 9536, 8584, 4956, 3466, 4166, 2952,
		9520, 9522, 8440, 3932, 3934, 7434, 8732, 3108, 9766, 12368, 10780,
		3160, 12102, 4688, 8604, 3070, 10402, 7500, 3206, 3588, 2636, 8452,
		11066, 9076, 7890, 8736, 10936, 10938:
		return 29
	case 1980, 7106, 8042:
		return 30
	case 2066, 4150, 11082, 11080, 9714, 11078, 3678, 10686:
		return 31
	case 2212, 5174, 5004, 5006, 5008, 8912:
		return 32
	case 2218:
		return 33
	case 2220:
		return 34
	case 2266:
		return 35
	case 2386:
		return 36
	case 2388:
		return 37
	case 2450:
		return 38
	case 2476, 4208, 12308, 10336, 9804:
		return 39
	case 4294, 1948:
		return 40
	case 2512, 9732, 6338, 6670, 3736, 10406, 10232, 10994, 7146:
		return 41
	case 2572, 11072:
		return 42
	case 2592, 9396, 2596, 10930, 9548, 9812, 9800, 5158:
		return 43
	case 2720:
		return 44
	case 2752:
		return 45
	case 2754, 9830, 9898:
		return 46
	case 14562:
		return 46
	case 2756:
		return 47
	case 2802:
		return 49
	case 2866:
		return 50
	case 2876:
		return 51
	case 2878, 2880:
		return 52
	case 2906, 4170, 2888, 4278, 4126:
		return 53
	case 2886:
		return 54
	case 2890:
		return 55
	case 2910:
		return 56
	case 3066, 10288:
		return 57
	case 3124, 5088:
		return 58
	case 3168, 3166:
		return 59
	case 3214, 9194, 4506:
		return 60
	case 7408, 3238:
		return 61
	case 3274:
		return 62
	case 3300:
		return 64
	case 3418:
		return 65
	case 3476:
		return 66
	case 3596:
		return 67
	case 3686:
		return 68
	case 3716, 6086:
		return 69
	case 4110, 2986, 4252:
		return 70
	case 4290:
		return 71
	case 4474:
		return 72
	case 4464, 9500:
		return 73
	case 4660:
		return 74
	case 4746, 4750, 4748:
		return 75
	case 4778, 6026, 7784:
		return 76
	case 4996:
		return 77
	case 4840:
		return 78
	case 5206:
		return 79
	case 5480, 9770, 9778, 9772, 9906, 9908, 9918, 10290, 10362:
		return 80
	case 6110:
		return 81
	case 6308:
		return 82
	case 6310:
		return 83
	case 6298:
		return 84
	case 6756:
		return 85
	case 7044:
		return 86
	case 6892:
		return 87
	case 6966:
		return 88
	case 7088, 11020:
		return 89
	case 7098, 9032:
		return 90
	case 10384:
		return 153
	case 7192:
		return 91
	case 7136, 11788:
		return 92
	case 7142:
		return 93
	case 7216:
		return 94
	case 7196:
		return 95
	case 7392, 9604:
		return 96
	case 6754:
		return 97
	case 7384:
		return 98
	case 7414:
		return 99
	case 7402, 7396:
		return 100
	case 7424:
		return 101
	case 7470, 9738:
		return 102
	case 7488:
		return 103
	case 7586, 7650:
		return 105
	case 6804, 6358, 7646:
		return 106
	case 7568, 7570, 7572, 7574:
		return 107
	case 7668:
		return 108
	case 7660, 9060:
		return 109
	case 7584:
		return 110
	case 7736, 9116, 9118, 7826, 7828, 11440, 11442, 11312, 7830, 7832,
		10670, 9120, 9122, 10680, 10626, 10578, 10334, 11380, 11326, 7912,
		11298, 10498, 7940, 12342, 8492, 9340, 11358:
		return 111
	case 7836, 7838, 7840, 7842:
		return 112
	case 7950:
		return 113
	case 8002:
		return 114
	case 8022:
		return 116
	case 8036:
		return 118
	case 9348, 8372, 8810:
		return 119
	case 8038, 11990, 8360, 8510, 8374:
		return 120
	case 8358:
		return 121
	case 8364:
		return 122
	case 8438:
		return 123
	case 10066, 8494, 11310, 9360:
		return 126
	case 8814:
		return 127
	case 8816, 8818, 8820, 8822:
		return 128
	case 8910:
		return 129
	case 8942:
		return 130
	case 8944, 5276, 10940:
		return 131
	case 8432, 8434, 8436, 8950:
		return 132
	case 8946, 9576, 9636:
		return 133
	case 8960:
		return 134
	case 9006:
		return 135
	case 9058, 13710:
		return 136
	case 9082, 9304, 9506:
		return 137
	case 9066:
		return 138
	case 9136:
		return 139
	case 9138:
		return 140
	case 9172, 9176:
		return 141
	case 9190:
		return 142
	case 9254:
		return 143
	case 9256:
		return 144
	case 9236:
		return 145
	case 9342:
		return 146
	case 9542:
		return 147
	case 9378:
		return 148
	case 9376:
		return 149
	case 9410:
		return 150
	case 9462:
		return 151
	case 9606, 9758:
		return 152
	case 9716, 5192, 9764, 9916:
		return 153
	case 10048, 9912:
		return 167
	case 10064, 10604:
		return 168
	case 10046:
		return 169
	case 10050:
		return 170
	case 10128:
		return 171
	case 10210, 9544:
		return 172
	case 10250:
		return 173
	case 10246:
		return 175
	case 10278:
		return 176
	case 10292, 7406, 9450:
		return 177
	case 10330:
		return 178
	case 10392:
		return 179
	case 10388, 9524, 9598:
		return 180
	case 11620:
		return 181
	case 10426:
		return 183
	case 10442:
		return 184
	case 10506:
		return 185
	case 10494:
		return 186
	case 10618:
		return 187
	case 10652:
		return 188
	case 10676:
		return 191
	case 10674:
		return 192
	case 10694:
		return 193
	case 10714:
		return 194
	case 10724:
		return 195
	case 10722:
		return 196
	case 10754:
		return 197
	case 10800:
		return 198
	case 10888:
		return 199
	case 10886:
		return 200
	case 10894:
		return 201
	case 10890:
		return 202
	case 9880, 9782, 9947, 10922, 9550, 9974:
		return 203
	case 10914:
		return 204
	case 10990:
		return 205
	case 10998:
		return 206
	case 10952:
		return 207
	case 11000:
		return 208
	case 11006:
		return 209
	case 11046:
		return 210
	case 11052:
		return 211
	case 10960:
		return 212
	case 10956, 9774, 9896, 10944:
		return 213
	case 10958:
		return 214
	case 10954:
		return 215
	case 11076:
		return 216
	case 11084, 10020:
		return 217
	case 11118, 9546, 9574, 9874, 9914:
		return 218
	case 11120:
		return 219
	case 11116:
		return 220
	case 11158:
		return 221
	case 11162:
		return 222
	case 11142:
		return 223
	case 11232:
		return 224
	case 11140:
		return 225
	case 11248, 9596:
		return 226
	case 11240:
		return 227
	case 11250:
		return 228
	case 11284:
		return 229
	case 11292:
		return 231
	case 11308:
		return 232
	case 11314:
		return 233
	case 11316:
		return 234
	case 11324:
		return 235
	case 11354:
		return 236
	case 11760, 11464, 11438, 12846, 12230, 11716, 11718, 11674, 11630,
		11786, 11872, 11762, 11994, 12172, 12184, 11460, 12014, 12016, 12018,
		12020, 12022, 12024, 12246, 12248, 12176, 12242, 11622, 12350, 12300,
		12374, 12356, 12286, 12628, 12420, 12384, 12410, 12412, 12404, 12402,
		12416, 12658, 11542, 12860, 12870, 12862, 12850, 12886, 12990, 12992,
		12880, 13060, 13136, 13114, 13118, 13190, 13326, 13330, 13332, 13366,
		13188, 13410, 13486, 13488, 13490, 13492, 13494, 13484, 13578, 13552,
		13554, 13572, 13606, 14302:
		return 237
	case 11384:
		return 239
	case 11458:
		return 240
	case 11814, 12232, 12302, 12872, 12874, 12958, 13324, 13424:
		return 241
	case 11548, 11552:
		return 242
	case 14538, 14540:
		return 242
	case 11704, 11706:
		return 243
	case 12180, 12346, 12344, 13058, 13498, 13322:
		return 244
	case 11506, 11508, 11562, 11768, 11882, 11720, 11884, 13116, 11536:
		return 245
	case 12432, 12434, 12842, 12640, 13268:
		return 246
	case 11818, 11876, 12000, 12240, 12642, 12644, 13022, 13024, 13396, 13398, 12564:
		return 248
	}
	return 0 // Default: no special punch effect
}
