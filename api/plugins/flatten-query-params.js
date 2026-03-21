// plugins/flatten-query-params.js
// Redocly decorator that flattens object query parameters into individual params.
//
// Usage in redocly.yaml:
//   plugins:
//     - "./plugins/flatten-query-params.js"
//   decorators:
//     plugin/flatten-query-params: on

/** @param {object} schema */
function isObjectSchema(schema) {
    if (!schema) return false;
    return (
        schema.type === 'object' ||
        Boolean(schema.properties) ||
        Boolean(schema.allOf)
    );
}

/**
 * Recursively resolves allOf and returns flat { properties, required }
 * @param {object} schema
 * @returns {{ properties: object, required: string[] }}
 */
function flattenSchema(schema, ctx) {
    if (!schema) return { properties: {}, required: [] };

    let properties = {};
    let required = [];

    if (schema.$ref) {
        const parts = schema.$ref.replace('#/', '').split('/');
        let resolved = ctx.document.parsed;
        for (const part of parts) {
            resolved = resolved[part];
        }
        return flattenSchema(resolved, ctx);
    }

    if (schema.allOf) {
        for (const sub of schema.allOf) {
            console.log(sub);
            const { properties: p, required: r } = flattenSchema(sub, ctx);
            properties = { ...properties, ...p };
            required = [...required, ...r];
        }
    }

    if (schema.properties) {
        properties = { ...properties, ...schema.properties };
    }

    if (Array.isArray(schema.required)) {
        required = [...required, ...schema.required];
    }

    return { properties, required };
}

/**
 * Expands a single object param into an array of flat params.
 * @param {object} param
 * @returns {object[]}
 */
function expandParam(param, ctx) {
    const schema = param.schema;
    if (!isObjectSchema(schema)) return [param];

    const { properties, required } = flattenSchema(schema, ctx);

    return Object.entries(properties).map(([name, propSchema]) => ({
        name,
        in: 'query',
        ...(required.includes(name) ? { required: true } : {}),
        schema: propSchema,
    }));
}

const FlattenQueryParams = () => ({
    Operation: {
        leave(operation, ctx) {
            if (!Array.isArray(operation.parameters)) return;

            const result = [];
            for (const param of operation.parameters) {
                const resolved = param.$ref ? ctx.resolve(param).node : param;

                if (resolved.in === 'query') {
                    result.push(...expandParam(resolved, ctx));
                } else {
                    result.push(param);
                }
            }

            operation.parameters = result;
        },
    },
});

module.exports = {
    id: 'plugin',
    decorators: {
        oas3: {
            'flatten-query-params': FlattenQueryParams,
        },
    },
};
