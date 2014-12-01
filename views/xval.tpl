{% extends "layout.tpl" %}

{% block content %}
    <div class="row">
        <div class="col-xs-12">
            <div class="page-header">
                <h1>Xbox 360 X Value Checker <small><a href="https://github.com/landaire/xval">Source Code</a></small></h1>
            </div>
        </div>
    </div>
    {% if has_errors %}
    <div class="row">
        <div class="col-xs-12">
            <div id="result" class="alert alert-danger">
                {% for index, errors in validation_errors %}
                    {% for error in errors %}
                    <p><strong>{{ index }}:</strong> {{error}}</p>
                    {% endfor %}
                {% endfor %}
            </div>
        </div>
    </div>
    {% endif %}
    {% if decryption_result %}
    <div class="row">
        <div class="col-xs-12">
            <div id="result" class="alert alert-success">
                <p>
                    <h3>X value decryption succeeded!</h3>
                    DES Key: {{ decryption_result.DesKey }}<br/>
                    Decrypted X value: {{ decryption_result.DecryptedData }}<br/>
                    {% if decryption_result.XValueFlags|length > 0 %}
                        <ul>
                        {% for flag in decryption_result.XValueFlags %}
                            <li>{{ flag }}</li>
                        {% endfor %}
                        </ul>
                    {% endif %}
                </p>
            </div>
        </div>
    </div>
    {% endif %}
    <div class="row">
        <div class="col-xs-12 col-sm-offset-4 col-sm-4">
            <div id="xvalForm" class="well">
                <form method="GET" action="/xval">
                    <div class="form-group">
                        <input type="text" class="form-control" name="serial" placeholder="Serial number" size="13" value="{{serial}}">
                    </div>
                    <div class="form-group">
                        <input type="text" class="form-control" name="xval" placeholder="X Value" size="20" value="{{xval}}">
                    </div>
                    <div class="form-group">
                        <button type="submit" class="btn btn-primary">Submit</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
{% endblock %}
