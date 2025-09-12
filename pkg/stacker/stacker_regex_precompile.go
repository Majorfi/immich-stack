package stacker

import (
	"fmt"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** PrecompileRegexes is a unified function that precompiles regex patterns from various
** criteria structures to warm the cache and avoid compilation cost during processing.
**
** @param criteriaSource - any that can be:
**   - []utils.TCriteria (legacy format)
**   - []utils.TCriteriaGroup (groups format)
**   - *utils.TCriteriaExpression (expression tree format)
**   - utils.TCriteria (single criteria)
** @return error - Compilation error if any regex pattern is invalid
**************************************************************************************************/
func PrecompileRegexes(criteriaSource any) error {
	switch source := criteriaSource.(type) {
	case []utils.TCriteria:
		for _, c := range source {
			if err := precompileCriteriaRegex(c); err != nil {
				return err
			}
		}
	case []utils.TCriteriaGroup:
		for _, g := range source {
			for _, c := range g.Criteria {
				if err := precompileCriteriaRegex(c); err != nil {
					return err
				}
			}
		}
	case *utils.TCriteriaExpression:
		return precompileExpressionRegexes(source)
	case utils.TCriteria:
		return precompileCriteriaRegex(source)
	default:
		return fmt.Errorf("unsupported criteria source type: %T", criteriaSource)
	}
	return nil
}

func precompileExpressionRegexes(expr *utils.TCriteriaExpression) error {
	if expr == nil {
		return nil
	}
	if expr.Criteria != nil {
		return precompileCriteriaRegex(*expr.Criteria)
	}
	for i := range expr.Children {
		if err := precompileExpressionRegexes(&expr.Children[i]); err != nil {
			return err
		}
	}
	return nil
}

func precompileCriteriaRegex(c utils.TCriteria) error {
	if c.Regex != nil && c.Regex.Key != "" {
		if _, err := utils.RegexCompile(c.Regex.Key); err != nil {
			return fmt.Errorf("failed to compile regex %q: %w", c.Regex.Key, err)
		}
	}
	return nil
}
