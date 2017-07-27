#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct Index Index;

Index* sparse_create();

void sparse_free(Index* index);

size_t sparse_iterator_size();

void sparse_begin(const Index* index, void* iterator);

int sparse_read_chunk(const Index* index, void* iterator, int64_t* start, int64_t* disk_start, int64_t* chunk_length);

int64_t sparse_read(const Index* index, int64_t start, int64_t* disk_start, int64_t* slice_length);

void sparse_write(Index* index, int64_t start, int64_t disk_start, int64_t slice_length);

#ifdef __cplusplus
}
#endif
