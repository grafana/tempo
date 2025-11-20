use datafusion::error::Result;
use datafusion::prelude::*;

/// SQL template for the flattened spans view
const SPANS_VIEW_SQL: &str = include_str!("sql/spans_view.sql");

/// Create the flattened spans view
///
/// This view unnests the nested trace structure and presents span attributes
/// as key-value pairs. This allows queries like:
/// ```sql
/// SELECT * FROM spans WHERE attr_key = 'http.scheme' AND attr_value = 'https'
/// ```
pub async fn create_flattened_view(ctx: &SessionContext) -> Result<()> {
    ctx.sql(SPANS_VIEW_SQL).await?;
    Ok(())
}
