{% extends "layout.tpl" %}

{% block content %}
    {{ body_content|markdown }}
{% endblock %}
