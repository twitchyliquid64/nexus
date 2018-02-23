


var app = angular.module('terminal', ['ui.materialize', 'angularMoment']);
var t1 = null;

t1 = new Terminal();
t1.setHeight("500px");
t1.setWidth('800px');
t1.print('Hi!');
document.getElementById("term-container").appendChild(t1.html);
