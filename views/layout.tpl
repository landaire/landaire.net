<!DOCTYPE html>
<html>
<head lang="en">
    <meta charset="UTF-8">
    <title>{{ title|default:"Lander's Things" }}</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.1/css/bootstrap.min.css">
    <link rel="stylesheet" href="/public/styles.css">
</head>
<body>
    <div class="container">
        {% block content %}

        {% endblock %}
    </div>

{% block javascript %}
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.1/js/bootstrap.min.js"></script>
{% endblock %}
</body>
</html>
