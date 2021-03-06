(function () {
  app.directive('datastoreFilter', function($rootScope){
    return {
      scope: {
        ds: '=',
        onChange: '&',
      },
      //restrict E means its can only be used as an element.
      restrict: 'E',
      templateUrl: function(elem, attr){
        return '/static/views/data/filterDirective.html?cache=2';
      },
      link: function($scope, elem, attrs) {
        // scope = either parent scope or its own child scope if scope set.
        // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
      },
      controller: function($scope) {
        $scope.filters = [];

        $scope.addFilterInput = '';
        $scope.addFilterConditional = '==';
        $scope.addFilterColumn = '!!';
        $scope.maxRows = 40;
        $scope.offset = 0;
        $scope.page = 0;

        $scope.getColName = function(colID){
          for (var i = 0; i < $scope.ds.Cols.length; i++){
            if ($scope.ds.Cols[i].UID == colID) return $scope.ds.Cols[i].Name;
          }
          return '?';
        }

        $scope.next = function(){
          $scope.page = ($scope.page + 1) % 2;
          console.log($scope.page);
        }

        $scope.setBounds = function(){
          $scope.onChange({filters: $scope.filters, limit: $scope.maxRows, offset: $scope.offset});
        }

        $scope.deleteFilter = function(index){
          $scope.filters.splice(index, 1);
          $scope.onChange({filters: $scope.filters, limit: $scope.maxRows, offset: $scope.offset});
        };

        $scope.addFilter = function(){
          if ($scope.addFilterColumn == '!!')return;

          if ($scope.addFilterInput.startsWith('{{')) {
            $scope.filters[$scope.filters.length] = {
              type: 'relativeConstraint',
              val: $scope.addFilterInput.slice(2,-2),
              col: $scope.addFilterColumn,
              conditional: $scope.addFilterConditional,
            }
          } else {
            $scope.filters[$scope.filters.length] = {
              type: 'literalConstraint',
              val: $scope.addFilterInput,
              col: $scope.addFilterColumn,
              conditional: $scope.addFilterConditional,
            }
          }
          $scope.addFilterInput = '';
          $scope.onChange({filters: $scope.filters, limit: $scope.maxRows, offset: $scope.offset});
        };


      },
    };
  });
})();
