(function () {
    app.directive('integrationDocumentationModal', function($rootScope, $sce){
      return {
        //scope allows us to setup variable bindings to the parent scope. By default, we share the parent scope. For an isolated one, we should
        //pass an object as the scope attribute which has a dict of the variable name for us, and a string describing where and how to bind it.
        //scope: {color: '@colorAttr'} binds 'color' to the value of color-attr with a one way binding. Only strings supported here: color-attr="{{color}}"
        //scope: {color: '=color'} binds 'color' two way. Pass the object here: color="color".
        scope: {

        },
        //restrict E means its can only be used as an element.
        restrict: 'E',
        templateUrl: function(elem, attr){
          return "/static/views/integration/documentationModal.html?cachebust=1"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.docs = $sce.trustAsHtml('');

          $rootScope.$on('documentation-modal', function(event, args) {
            $scope.open = true;
            $scope.docs = $sce.trustAsHtml(args.docs);
          });

          $scope.close = function(){
            $scope.open = false;
          }
        }
      };
  });
})();
