{% extends "layout.tpl" %}

{% block content %}
    <div class="col-xs-12">
        <div class="row">
            <div class="alert alert-dismissable alert-info">
                <button type="button" class="close" data-dismiss="alert">×</button>
                <strong>Heads up!</strong> My website is undergoing reworking. Don't judge me on layout!
            </div>
        </div>
    </div>
    <div class="col-xs-12">
        <div class="row">
            {{ body_content|markdown }}
        </div>
    </div>
{% endblock %}
