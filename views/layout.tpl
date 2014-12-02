<!DOCTYPE html>
<html>
<head lang="en">
    <meta charset="UTF-8">
    <title>{{ title|default:"Lander's Things" }}</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.1/css/bootstrap.min.css">
    <link rel="stylesheet" href="/styles.css">
</head>
<body>
    <div class="container">
        {% if show_back_link|default:false %}
            <div class="row">
                <div class="col-xs-3">
                    <a href="/">&larr; Back to main page</a>
                </div>
            </div>
        {% endif %}
        {% block content %}

        {% endblock %}
    </div>

{% block javascript %}
    <script src="//cdnjs.cloudflare.com/ajax/libs/jquery/2.1.1/jquery.min.js"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.1/js/bootstrap.min.js"></script>
{% endblock %}
</body>
</html>
