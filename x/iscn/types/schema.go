package types

var PeriodSchema = NewSchemaValidator(
	Field("start", InType(None, String)),
	Field("end", InType(None, String)),
)

var EntitySchema = NewSchemaValidator(
	Field("id", InType(String)),
	Field("name", InType(None, String)),
	Field("description", InType(None, String)),
)

var RightSchema = NewSchemaValidator(
	Field("holder", Any(
		IsCIDWithCodec(EntityCodecType),
		InSchema(EntitySchema),
	)),
	Field("type", InType(String)),
	Field("terms", IsCIDWithCodec(RightTermsCodecType)),
	Field("period", Any(
		InType(None),
		InSchema(PeriodSchema),
	)),
	Field("territory", InType(None, String)),
)

var RightsSchema = NewSchemaValidator(
	Field("rights", IsArrayOf(InSchema(RightSchema))),
)

var StakeholderSchema = NewSchemaValidator(
	Field("stakeholder", Any(
		IsCIDWithCodec(EntityCodecType),
		InSchema(EntitySchema),
	)),
	Field("type", InType(String)),
	Field("sharing", IsUint32),
	Field("footprint", InType(None, String)),
)

var StakeholdersSchema = NewSchemaValidator(
	Field("stakeholders", IsArrayOf(InSchema(StakeholderSchema))),
)

var ContentSchema = NewSchemaValidator(
	Field("type", InType(String)),
	Field("version", IsUint64),
	Field("parent", Any(
		InType(None),
		IsCIDWithCodec(IscnContentCodecType),
	)),
	Field("source", Any(
		InType(None),
		IsURL,
	)),
	Field("edition", InType(None, String)),
	Field("fingerprint", IsURL),
	Field("title", InType(String)),
	Field("description", InType(None, String)),
	Field("tags", Any(
		InType(None),
		IsArrayOf(InType(String)),
	)),
)

var KernelSchema = NewSchemaValidator(
	Field("timestamp", IsTimestamp),
	Field("version", IsUint64),
	Field("parent", Any(
		InType(None),
		IsCIDWithCodec(IscnKernelCodecType),
	)),
	Field("rights", InSchema(RightsSchema)),
	Field("stakeholders", InSchema(StakeholdersSchema)),
	Field("content", InSchema(ContentSchema)),
)
