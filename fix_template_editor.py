import os

file_path = 'frontend/src/components/TemplateEditor.tsx'

with open(file_path, 'r') as f:
    content = f.read()

# Найти функцию findVariables и убедиться, что она правильно извлекает имена
# Найти блок рендеринга переменных и убедиться, что onChange обновляет context

# Простое решение: добавить принудительный рендеринг с логированием
new_content = content.replace(
    'const handleRender = async () => {',
    '''  const handleRender = async () => {
    console.log("Template:", template);
    console.log("Context:", context);
    // Ensure all variables have values
    const vars = findVariables(template);
    const missing = vars.filter(v => !(v in context) || context[v] === "");
    if (missing.length > 0) {
      setError("Missing values for variables: " + missing.join(", "));
      setLoading(false);
      return;
    }'''
)

if new_content != content:
    with open(file_path, 'w') as f:
        f.write(new_content)
    print("✅ Updated TemplateEditor.tsx with logging and validation")
else:
    print("⚠️ Could not update TemplateEditor.tsx")
